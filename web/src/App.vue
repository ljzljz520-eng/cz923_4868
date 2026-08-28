<template>
  <main class="workspace">
    <header class="topbar">
      <div>
        <p class="facility">门诊药房 · 发药区</p>
        <h1>单体药房取药台</h1>
      </div>
      <div class="top-actions">
        <span class="status"><i></i>服务正常</span>
        <button class="icon-button" title="刷新列表" aria-label="刷新列表" @click="load">↻</button>
        <button class="primary" @click="showRegister = true">登记取药</button>
      </div>
    </header>

    <section class="summary" aria-label="队列摘要">
      <div><strong>{{ dashboard.waitingCount }}</strong><span>待取药</span></div>
      <div><strong>{{ dashboard.calledCount }}</strong><span>已叫号</span></div>
      <div><strong>{{ dashboard.completedCount }}</strong><span>今日完成</span></div>
      <label class="search"><span>搜索</span><input v-model="query" placeholder="姓名、票号或处方号"></label>
    </section>

    <p v-if="message" :class="['message', messageType]">{{ message }}</p>

    <section class="board">
      <QueueColumn title="待取药" tone="waiting" :orders="filteredWaiting" empty="暂无待取药患者">
        <template #actions="{ order }">
          <button class="primary small" @click="callOrder(order)">叫号</button>
        </template>
      </QueueColumn>
      <QueueColumn title="已叫号" tone="called" :orders="filteredCalled" empty="暂无已叫号患者">
        <template #actions="{ order }">
          <button class="primary small" @click="openDispense(order)">核验发药</button>
        </template>
      </QueueColumn>
      <QueueColumn title="已完成" tone="completed" :orders="filteredCompleted" empty="今日暂无完成记录">
        <template #actions="{ order }">
          <span class="done">已完成</span>
        </template>
      </QueueColumn>
    </section>

    <div v-if="showRegister" class="overlay" @click.self="showRegister = false">
      <form class="dialog" @submit.prevent="registerOrder">
        <header><div><p>新建业务</p><h2>登记取药单</h2></div><button type="button" class="icon-button" aria-label="关闭" @click="showRegister = false">×</button></header>
        <div class="form-grid">
          <label>取药票号<input v-model.trim="form.ticketNumber" required></label>
          <label>患者姓名<input v-model.trim="form.patientName" required></label>
          <label>患者编号<input v-model.trim="form.patientCode" required></label>
          <label>处方编号<input v-model.trim="form.prescriptionId" required></label>
          <label>优先级<select v-model="form.priority"><option value="routine">普通</option><option value="senior">优先</option><option value="urgent">紧急</option></select></label>
          <label>药品<select v-model="form.medicineCode"><option v-for="medicine in medicines" :key="medicine.code" :value="medicine.code">{{ medicine.name }}</option></select></label>
          <label>批号<input v-model.trim="form.lot" required></label>
          <label>数量<input v-model.number="form.quantity" type="number" min="1" required></label>
        </div>
        <footer><button type="button" class="secondary" @click="showRegister = false">取消</button><button class="primary" type="submit">确认登记</button></footer>
      </form>
    </div>

    <div v-if="dispensing" class="overlay" @click.self="dispensing = null">
      <form class="dialog" @submit.prevent="completeDispense">
        <header><div><p>处方核验</p><h2>{{ dispensing.ticketNumber }} · {{ dispensing.patientName }}</h2></div><button type="button" class="icon-button" aria-label="关闭" @click="dispensing = null">×</button></header>
        <div class="verification" v-for="item in dispensing.items" :key="item.id">
          <div><strong>{{ item.name }}</strong><span>{{ item.specification }} · {{ item.quantity }}{{ item.unit }}</span></div>
          <label><input type="checkbox" v-model="checks[item.medicineCode]"> 药品、批号与数量一致</label>
        </div>
        <label class="note">发药备注<textarea v-model.trim="dispenseNote" rows="3"></textarea></label>
        <footer><button type="button" class="secondary" @click="dispensing = null">返回</button><button class="primary" type="submit">完成发药</button></footer>
      </form>
    </div>
  </main>
</template>

<script>
import { api } from './api'

const QueueColumn = {
  props: ['title', 'tone', 'orders', 'empty'],
  template: `<section class="column"><header><div><i :class="tone"></i><h2>{{ title }}</h2></div><span>{{ orders.length }}</span></header><div class="list"><article v-for="order in orders" :key="order.id" class="order"><div class="order-head"><strong>{{ order.ticketNumber }}</strong><span :class="['priority', order.priority]">{{ priority(order.priority) }}</span></div><h3>{{ order.patientName }}</h3><p>{{ order.patientCode }} · {{ order.prescriptionId }}</p><ul><li v-for="item in order.items" :key="item.id">{{ item.name }} <span>{{ item.quantity }}{{ item.unit }}</span></li></ul><footer><time>{{ formatTime(order.updatedAt) }}</time><slot name="actions" :order="order"></slot></footer></article><div v-if="orders.length === 0" class="empty">{{ empty }}</div></div></section>`,
  methods: {
    priority(value) { return { routine: '普通', senior: '优先', urgent: '紧急' }[value] || value },
    formatTime(value) { return new Date(value).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }) }
  }
}

export default {
  components: { QueueColumn },
  data() {
    return {
      dashboard: { waiting: [], called: [], completed: [], waitingCount: 0, calledCount: 0, completedCount: 0 },
      medicines: [], query: '', message: '', messageType: 'success', showRegister: false, dispensing: null, checks: {}, dispenseNote: '',
      form: { ticketNumber: '', patientName: '', patientCode: '', prescriptionId: '', priority: 'routine', medicineCode: 'MED001', lot: '', quantity: 1 }
    }
  },
  computed: {
    filteredWaiting() { return this.filter(this.dashboard.waiting) },
    filteredCalled() { return this.filter(this.dashboard.called) },
    filteredCompleted() { return this.filter(this.dashboard.completed) }
  },
  async mounted() { await Promise.all([this.load(), this.loadMedicines()]) },
  methods: {
    filter(orders) { const q = this.query.toLowerCase(); return orders.filter(o => !q || `${o.patientName} ${o.ticketNumber} ${o.prescriptionId}`.toLowerCase().includes(q)) },
    notify(text, type = 'success') { this.message = text; this.messageType = type; window.setTimeout(() => { this.message = '' }, 3500) },
    async load() { try { this.dashboard = await api.dashboard() } catch (error) { this.notify(error.message, 'error') } },
    async loadMedicines() { try { this.medicines = await api.medicines() } catch (error) { this.notify(error.message, 'error') } },
    async registerOrder() {
      const medicine = this.medicines.find(item => item.code === this.form.medicineCode)
      const now = new Date().toISOString()
      const command = { id: `order-${this.form.prescriptionId}`, ...this.form, createdAt: now, items: [{ medicineCode: medicine.code, name: medicine.name, specification: medicine.specification, unit: medicine.unit, location: medicine.location, quantity: this.form.quantity, lot: this.form.lot }] }
      try { await api.register(command); this.showRegister = false; this.notify('取药单已登记'); await this.load() } catch (error) { this.notify(error.message, 'error') }
    },
    async callOrder(order) {
      try { await api.call(order.id, { orderId: order.id, counterCode: 'C01', operator: 'web-pharmacist', calledAt: new Date().toISOString() }); this.notify(`${order.ticketNumber} 已叫号`); await this.load() } catch (error) { this.notify(error.message, 'error') }
    },
    openDispense(order) { this.dispensing = order; this.checks = {}; order.items.forEach(item => { this.$set(this.checks, item.medicineCode, false) }); this.dispenseNote = '' },
    async completeDispense() {
      const order = this.dispensing
      const checks = order.items.map(item => ({ medicineCode: item.medicineCode, lot: item.lot, quantity: item.quantity, confirmed: Boolean(this.checks[item.medicineCode]) }))
      try { await api.dispense(order.id, { recordId: `dispense-${order.id}`, counterCode: 'C01', operator: 'web-pharmacist', completedAt: new Date().toISOString(), note: this.dispenseNote, checks }); this.dispensing = null; this.notify('发药已完成'); await this.load() } catch (error) { this.notify(error.message, 'error') }
    }
  }
}
</script>
