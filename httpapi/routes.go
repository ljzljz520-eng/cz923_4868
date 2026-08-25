package httpapi

import (
	"net/http"
	"strings"

	"pharmacycounter/catalog"
	"pharmacycounter/domain"
	"pharmacycounter/query"
)

func (s *Server) handleHealth(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		methodAllowed(writer, http.MethodGet)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDashboard(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		methodAllowed(writer, http.MethodGet)
		return
	}
	dashboard, err := s.queries.Dashboard()
	if err != nil {
		writeError(writer, statusForError(err), err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, dashboard)
}

func (s *Server) handleOrders(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		filter, err := orderFilter(request)
		if err != nil {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		orders, err := s.queries.Orders(filter)
		if err != nil {
			writeError(writer, statusForError(err), err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, orders)
	case http.MethodPost:
		var command domain.CreateOrderCommand
		if err := decodeJSON(request, &command); err != nil {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		command.Items = catalog.FillCatalogDetails(s.catalog, command.Items)
		if err := s.catalog.ValidateItems(command.Items); err != nil {
			writeError(writer, http.StatusUnprocessableEntity, err.Error())
			return
		}
		order, err := s.business.Register(command, request.Header.Get("X-Operator"))
		if err != nil {
			writeError(writer, statusForError(err), err.Error())
			return
		}
		for _, warning := range s.catalog.Warnings(order.Items) {
			order, err = s.business.AddWarning(order.ID, warning, request.Header.Get("X-Operator"))
			if err != nil {
				writeError(writer, statusForError(err), err.Error())
				return
			}
		}
		writeJSON(writer, http.StatusCreated, order)
	default:
		methodAllowed(writer, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) handleOrderAction(writer http.ResponseWriter, request *http.Request) {
	path := strings.TrimPrefix(request.URL.Path, "/api/orders/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && request.Method == http.MethodGet {
		order, entries, err := s.queries.OrderDetail(parts[0])
		if err != nil {
			writeError(writer, statusForError(err), err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"order": order, "audit": entries})
		return
	}
	if len(parts) != 2 || request.Method != http.MethodPost {
		methodAllowed(writer, http.MethodGet, http.MethodPost)
		return
	}
	switch parts[1] {
	case "call":
		var command domain.CallCommand
		if err := decodeJSON(request, &command); err != nil {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		command.OrderID = parts[0]
		order, record, err := s.business.CallNext(command)
		if err != nil {
			writeError(writer, statusForError(err), err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"order": order, "call": record})
	case "dispense":
		var command domain.DispenseCommand
		if err := decodeJSON(request, &command); err != nil {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		command.OrderID = parts[0]
		order, record, err := s.business.CompleteDispense(command)
		if err != nil {
			writeError(writer, statusForError(err), err.Error())
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"order": order, "dispense": record})
	default:
		writeError(writer, http.StatusNotFound, "操作不存在")
	}
}

func (s *Server) handleMedicines(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		methodAllowed(writer, http.MethodGet)
		return
	}
	writeJSON(writer, http.StatusOK, s.catalog.Search(request.URL.Query().Get("q")))
}

func (s *Server) handleDailyReport(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		methodAllowed(writer, http.MethodGet)
		return
	}
	orders, err := s.queries.Orders(domain.OrderFilter{})
	if err != nil {
		writeError(writer, statusForError(err), err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, query.BuildDailySummaries(orders))
}

func (s *Server) handleMedicineReport(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		methodAllowed(writer, http.MethodGet)
		return
	}
	orders, err := s.queries.Orders(domain.OrderFilter{})
	if err != nil {
		writeError(writer, statusForError(err), err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, query.BuildMedicineUsage(orders))
}
