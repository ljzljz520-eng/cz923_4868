package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pharmacycounter/domain"
)

type responseEnvelope struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(responseEnvelope{Data: value})
}

func writeError(writer http.ResponseWriter, status int, message string) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(responseEnvelope{Error: message})
}

func decodeJSON(request *http.Request, target any) error {
	if request.Body == nil {
		return errors.New("请求体不能为空")
	}
	decoder := json.NewDecoder(io.LimitReader(request.Body, 1024*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errors.New("请求体只能包含一个 JSON 对象")
	}
	return nil
}

func statusForError(err error) int {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrDuplicate), errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrInvalidTransition):
		return http.StatusConflict
	case errors.Is(err, domain.ErrValidation), errors.Is(err, domain.ErrUnresolvedWarning):
		return http.StatusUnprocessableEntity
	default:
		var validation domain.ValidationError
		if errors.As(err, &validation) {
			return http.StatusUnprocessableEntity
		}
		return http.StatusInternalServerError
	}
}

func parseStates(value string) ([]domain.OrderState, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	states := make([]domain.OrderState, 0, len(parts))
	for _, part := range parts {
		state := domain.OrderState(strings.TrimSpace(part))
		if err := domain.ValidateState(state); err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	return states, nil
}

func parseInteger(value string, fallback int) (int, error) {
	if strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func parseTime(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, value)
}

func orderFilter(request *http.Request) (domain.OrderFilter, error) {
	query := request.URL.Query()
	states, err := parseStates(query.Get("state"))
	if err != nil {
		return domain.OrderFilter{}, err
	}
	limit, err := parseInteger(query.Get("limit"), 100)
	if err != nil {
		return domain.OrderFilter{}, err
	}
	offset, err := parseInteger(query.Get("offset"), 0)
	if err != nil {
		return domain.OrderFilter{}, err
	}
	from, err := parseTime(query.Get("from"))
	if err != nil {
		return domain.OrderFilter{}, err
	}
	to, err := parseTime(query.Get("to"))
	if err != nil {
		return domain.OrderFilter{}, err
	}
	priority := domain.Priority(query.Get("priority"))
	if priority != "" {
		if err := domain.ValidatePriority(priority); err != nil {
			return domain.OrderFilter{}, err
		}
	}
	return domain.OrderFilter{States: states, PatientQuery: query.Get("q"), Priority: priority, CreatedFrom: from, CreatedTo: to, Limit: limit, Offset: offset}, nil
}

func methodAllowed(writer http.ResponseWriter, allowed ...string) {
	writer.Header().Set("Allow", strings.Join(allowed, ", "))
	writeError(writer, http.StatusMethodNotAllowed, "请求方法不支持")
}
