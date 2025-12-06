<template>
    <div class="container">
        <header>
            <h1>🏦 Bank AML System</h1>
            <p class="subtitle">Система противодействия отмыванию денег и мошенничеству</p>
        </header>

        <div class="status-bar">
            <div class="status-item" :class="{ 'online': ingestionStatus, 'offline': !ingestionStatus }">
                <span class="status-dot"></span>
                Сервис приема транзакций: {{ ingestionStatus ? 'Онлайн' : 'Офлайн' }}
            </div>
            <div class="status-item" :class="{ 'online': fraudStatus, 'offline': !fraudStatus }">
                <span class="status-dot"></span>
                Сервис детекции мошенничества: {{ fraudStatus ? 'Онлайн' : 'Офлайн' }}
            </div>
        </div>

        <!-- Вкладки -->
        <div class="tabs">
            <button 
                @click="activeTab = 'transactions'" 
                class="tab-button"
                :class="{ 'active': activeTab === 'transactions' }"
            >
                💳 Транзакции
            </button>
            <button 
                @click="activeTab = 'logs'" 
                class="tab-button"
                :class="{ 'active': activeTab === 'logs' }"
            >
                📊 Логи и Аналитика
            </button>
        </div>

        <div class="main-content" v-if="activeTab === 'transactions'">
            <!-- Форма отправки транзакции -->
            <section class="card">
                <h2>📝 Отправить транзакцию</h2>
                <form @submit.prevent="submitTransaction" class="transaction-form">
                    <div class="form-row">
                        <div class="form-group">
                            <label>ID транзакции *</label>
                            <input v-model="form.transaction_id" type="text" required placeholder="TXN-001">
                        </div>
                        <div class="form-group">
                            <label>Номер счета *</label>
                            <input v-model="form.account_number" type="text" required placeholder="ACC123456789">
                        </div>
                    </div>

                    <div class="form-row">
                        <div class="form-group">
                            <label>Сумма *</label>
                            <input v-model.number="form.amount" type="number" step="0.01" required placeholder="1000.00">
                        </div>
                        <div class="form-group">
                            <label>Валюта *</label>
                            <select v-model="form.currency" required>
                                <option value="USD">USD</option>
                                <option value="EUR">EUR</option>
                                <option value="RUB">RUB</option>
                                <option value="GBP">GBP</option>
                            </select>
                        </div>
                    </div>

                    <div class="form-row">
                        <div class="form-group">
                            <label>Тип транзакции *</label>
                            <select v-model="form.transaction_type" required>
                                <option value="transfer">Перевод</option>
                                <option value="international_transfer">Международный перевод</option>
                                <option value="withdrawal">Снятие</option>
                                <option value="deposit">Пополнение</option>
                            </select>
                        </div>
                        <div class="form-group">
                            <label>Канал *</label>
                            <select v-model="form.channel" required>
                                <option value="online">Онлайн</option>
                                <option value="mobile">Мобильное приложение</option>
                                <option value="branch">Отделение банка</option>
                                <option value="atm">Банкомат</option>
                            </select>
                        </div>
                    </div>

                    <div class="form-row">
                        <div class="form-group">
                            <label>Счет контрагента</label>
                            <input v-model="form.counterparty_account" type="text" placeholder="ACC987654321">
                        </div>
                        <div class="form-group">
                            <label>Банк контрагента</label>
                            <input v-model="form.counterparty_bank" type="text" placeholder="Test Bank">
                        </div>
                    </div>

                    <div class="form-row">
                        <div class="form-group">
                            <label>Страна контрагента</label>
                            <select v-model="form.counterparty_country">
                                <option value="">Выберите страну</option>
                                <option value="US">США</option>
                                <option value="GB">Великобритания</option>
                                <option value="CH">Швейцария</option>
                                <option value="RU">Россия</option>
                                <option value="KY">Каймановы острова</option>
                                <option value="VG">Британские Виргинские острова</option>
                            </select>
                        </div>
                        <div class="form-group">
                            <label>ID пользователя</label>
                            <input v-model="form.user_id" type="text" placeholder="user123">
                        </div>
                    </div>

                    <div class="form-group">
                        <label>ID отделения</label>
                        <input v-model="form.branch_id" type="text" placeholder="branch001">
                    </div>

                    <button type="submit" class="btn-primary" :disabled="loading">
                        {{ loading ? 'Отправка...' : '🚀 Отправить транзакцию' }}
                    </button>
                </form>

                <div class="form-actions">
                    <h3>⚡ Быстрые действия</h3>
                    <div class="quick-actions">
                        <div class="action-group">
                            <button 
                                @click="generateRandomTransaction" 
                                class="btn-generate btn-random"
                                :disabled="loading"
                            >
                                🎲 Автогенерация случайных данных
                            </button>
                        </div>
                        <div class="action-group">
                            <button 
                                @click="clearDatabase" 
                                class="btn-danger"
                                :disabled="loading"
                            >
                                🗑️ Очистить все транзакции
                            </button>
                        </div>
                    </div>
                </div>
            </section>

            <!-- Список транзакций -->
            <section class="card">
                <div class="card-header">
                    <h2>📊 История транзакций</h2>
                    <button @click="refreshTransactions" class="btn-refresh" :disabled="loading">
                        🔄 Обновить
                    </button>
                </div>

                <div v-if="transactions.length === 0" class="empty-state">
                    <p>Нет транзакций. Отправьте первую транзакцию выше.</p>
                </div>

                <div v-else class="transactions-list">
                    <div 
                        v-for="tx in transactions" 
                        :key="tx.processing_id" 
                        class="transaction-item"
                        :class="getRiskClass(tx.risk_level)"
                        @click="selectTransaction(tx)"
                    >
                        <div class="transaction-header">
                            <div class="transaction-id">
                                <strong>{{ tx.transaction_id }}</strong>
                                <span class="processing-id">{{ tx.processing_id }}</span>
                            </div>
                            <div class="transaction-status">
                                <span class="status-badge" :class="getStatusClass(tx.status)">
                                    {{ getStatusText(tx.status) }}
                                </span>
                            </div>
                        </div>

                        <div class="transaction-details">
                            <div class="detail-item" v-if="tx.amount !== null && tx.amount !== undefined">
                                <span class="label">Сумма:</span>
                                <span class="value">{{ formatAmount(tx.amount, tx.currency || 'USD') }}</span>
                            </div>
                            <div class="detail-item" v-if="tx.risk_score !== null">
                                <span class="label">Оценка риска:</span>
                                <span class="value risk-score" :class="getRiskClass(tx.risk_level)">
                                    {{ tx.risk_score }}
                                </span>
                            </div>
                            <div class="detail-item" v-if="tx.risk_level">
                                <span class="label">Уровень риска:</span>
                                <span class="value risk-level" :class="getRiskClass(tx.risk_level)">
                                    {{ getRiskLevelText(tx.risk_level) }}
                                </span>
                            </div>
                        </div>

                        <div v-if="tx.flags && tx.flags.length > 0" class="transaction-flags">
                            <span v-for="flag in tx.flags" :key="flag" class="flag-badge">
                                {{ getFlagText(flag) }}
                            </span>
                        </div>
                    </div>
                </div>
            </section>

            <!-- Детали выбранной транзакции -->
            <section v-if="selectedTransaction" class="card details-card">
                <div class="card-header">
                    <h2>🔍 Детали транзакции</h2>
                    <button @click="selectedTransaction = null" class="btn-close">✕</button>
                </div>

                <div class="transaction-details-full">
                    <div class="detail-section">
                        <h3>Основная информация</h3>
                        <div class="detail-grid">
                            <div class="detail-row">
                                <span class="detail-label">ID транзакции:</span>
                                <span class="detail-value">{{ selectedTransaction.transaction_id }}</span>
                            </div>
                            <div class="detail-row">
                                <span class="detail-label">ID обработки:</span>
                                <span class="detail-value">{{ selectedTransaction.processing_id }}</span>
                            </div>
                            <div class="detail-row">
                                <span class="detail-label">Номер счета:</span>
                                <span class="detail-value">{{ selectedTransaction.account_number }}</span>
                            </div>
                            <div class="detail-row" v-if="selectedTransaction.amount !== null && selectedTransaction.amount !== undefined">
                                <span class="detail-label">Сумма:</span>
                                <span class="detail-value">{{ formatAmount(selectedTransaction.amount, selectedTransaction.currency || 'USD') }}</span>
                            </div>
                            <div class="detail-row">
                                <span class="detail-label">Статус:</span>
                                <span class="detail-value status-badge" :class="getStatusClass(selectedTransaction.status)">
                                    {{ getStatusText(selectedTransaction.status) }}
                                </span>
                            </div>
                        </div>
                    </div>

                    <div v-if="selectedTransaction.risk_score !== null" class="detail-section">
                        <h3>Анализ рисков</h3>
                        <div class="detail-grid">
                            <div class="detail-row">
                                <span class="detail-label">Оценка риска:</span>
                                <span class="detail-value risk-score-large" :class="getRiskClass(selectedTransaction.risk_level)">
                                    {{ selectedTransaction.risk_score }}
                                </span>
                            </div>
                            <div class="detail-row">
                                <span class="detail-label">Уровень риска:</span>
                                <span class="detail-value risk-level-large" :class="getRiskClass(selectedTransaction.risk_level)">
                                    {{ selectedTransaction.risk_level ? getRiskLevelText(selectedTransaction.risk_level) : 'N/A' }}
                                </span>
                            </div>
                            <div class="detail-row" v-if="selectedTransaction.analysis_timestamp">
                                <span class="detail-label">Время анализа:</span>
                                <span class="detail-value">{{ formatDate(selectedTransaction.analysis_timestamp) }}</span>
                            </div>
                        </div>
                    </div>

                    <div v-if="selectedTransaction.flags && selectedTransaction.flags.length > 0" class="detail-section">
                        <h3>Флаги риска</h3>
                        <div class="flags-container">
                            <span v-for="flag in selectedTransaction.flags" :key="flag" class="flag-badge-large">
                                {{ getFlagText(flag) }}
                            </span>
                        </div>
                    </div>
                </div>
            </section>
        </div>

        <!-- Вкладка логов и аналитики -->
        <div class="main-content" v-if="activeTab === 'logs'">
            <!-- Статистика -->
            <section class="card">
                <h2>📈 Статистика системы</h2>
                <div class="stats-grid">
                    <div class="stat-card">
                        <div class="stat-value">{{ stats.total_events || 0 }}</div>
                        <div class="stat-label">Всего событий</div>
                    </div>
                    <div class="stat-card" v-for="(count, component) in stats.components" :key="component">
                        <div class="stat-value">{{ count }}</div>
                        <div class="stat-label">{{ getComponentName(component) }}</div>
                    </div>
                </div>

                <div class="stats-breakdown">
                    <h3>По сервисам:</h3>
                    <div class="stats-list">
                        <div v-for="(count, service) in stats.services" :key="service" class="stat-item">
                            <span class="stat-service">{{ getServiceName(service) }}:</span>
                            <span class="stat-count">{{ count }}</span>
                        </div>
                    </div>
                </div>

                <div class="stats-breakdown">
                    <h3>По типам событий:</h3>
                    <div class="stats-list">
                        <div v-for="(count, eventType) in stats.event_types" :key="eventType" class="stat-item">
                            <span class="stat-service">{{ getEventTypeName(eventType) }}:</span>
                            <span class="stat-count">{{ count }}</span>
                        </div>
                    </div>
                </div>
            </section>

            <!-- Логи событий -->
            <section class="card">
                <div class="card-header">
                    <h2>📋 Логи событий (последние 100)</h2>
                    <button @click="refreshLogs" class="btn-refresh" :disabled="loading">
                        🔄 Обновить
                    </button>
                </div>

                <div class="logs-container">
                    <div 
                        v-for="event in events" 
                        :key="event.id" 
                        class="log-entry"
                        :class="getLogClass(event)"
                    >
                        <div class="log-header">
                            <span class="log-time">{{ formatTime(event.timestamp) }}</span>
                            <span class="log-service">{{ getServiceName(event.service) }}</span>
                            <span class="log-component" :class="getComponentClass(event.component)">
                                {{ getComponentName(event.component) }}
                            </span>
                            <span class="log-type">{{ getEventTypeName(event.type) }}</span>
                        </div>
                        <div class="log-data">
                            <div v-for="(value, key) in event.data" :key="key" class="log-data-item">
                                <strong>{{ key }}:</strong> {{ formatLogValue(value) }}
                            </div>
                        </div>
                    </div>
                </div>

                <div v-if="events.length === 0" class="empty-state">
                    <p>Нет событий. Отправьте транзакцию, чтобы увидеть логи.</p>
                </div>
            </section>
        </div>

        <!-- Уведомления -->
        <div v-if="notification" class="notification" :class="notification.type">
            {{ notification.message }}
        </div>
    </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

const ingestionStatus = ref(false)
const fraudStatus = ref(false)
const loading = ref(false)
const transactions = ref([])
const selectedTransaction = ref(null)
const notification = ref(null)
const activeTab = ref('transactions')
const events = ref([])
const stats = ref({})

const form = ref({
    transaction_id: '',
    account_number: '',
    amount: '',
    currency: 'USD',
    transaction_type: 'transfer',
    counterparty_account: '',
    counterparty_bank: '',
    counterparty_country: '',
    channel: 'online',
    user_id: '',
    branch_id: ''
})

const checkServices = async () => {
    try {
        await axios.get('http://localhost:8080/health', { timeout: 1000 })
        ingestionStatus.value = true
    } catch (e) {
        ingestionStatus.value = false
    }

    try {
        await axios.get('http://localhost:8081/health', { timeout: 1000 })
        fraudStatus.value = true
    } catch (e) {
        fraudStatus.value = false
    }
}

const submitTransaction = async () => {
    loading.value = true
    try {
        const response = await axios.post('http://localhost:8080/api/v1/transactions', {
            transaction_id: form.value.transaction_id,
            account_number: form.value.account_number,
            amount: parseFloat(form.value.amount),
            currency: form.value.currency,
            transaction_type: form.value.transaction_type,
            counterparty_account: form.value.counterparty_account || null,
            counterparty_bank: form.value.counterparty_bank || null,
            counterparty_country: form.value.counterparty_country || null,
            channel: form.value.channel,
            user_id: form.value.user_id || null,
            branch_id: form.value.branch_id || null,
            timestamp: new Date().toISOString()
        })

        showNotification('Транзакция успешно отправлена!', 'success')
        
        form.value = {
            transaction_id: '',
            account_number: '',
            amount: '',
            currency: 'USD',
            transaction_type: 'transfer',
            counterparty_account: '',
            counterparty_bank: '',
            counterparty_country: '',
            channel: 'online',
            user_id: '',
            branch_id: ''
        }

        setTimeout(() => loadTransactions(), 2000)
    } catch (error) {
        showNotification('Ошибка при отправке транзакции: ' + (error.response?.data?.error || error.message), 'error')
    } finally {
        loading.value = false
    }
}

const loadTransactions = async () => {
    try {
        const response = await axios.get('http://localhost:8080/api/v1/transactions?limit=50')
        transactions.value = response.data.transactions || []
    } catch (error) {
        console.error('Error loading transactions:', error)
        // Fallback на старый метод если новый не работает
        const savedIds = JSON.parse(localStorage.getItem('transaction_ids') || '[]')
        if (savedIds.length > 0) {
            const promises = savedIds.slice(-10).map(id => 
                axios.get(`http://localhost:8080/api/v1/transactions/${id}`)
                    .then(res => res.data)
                    .catch(() => null)
            )
            const results = await Promise.all(promises)
            transactions.value = results.filter(tx => tx !== null).reverse()
        }
    }
}

const refreshTransactions = async () => {
    loading.value = true
    await loadTransactions()
    loading.value = false
    showNotification('Список транзакций обновлен', 'success')
}

const selectTransaction = async (tx) => {
    try {
        const response = await axios.get(`http://localhost:8081/api/v1/transactions/${tx.processing_id}`)
            .catch(() => axios.get(`http://localhost:8080/api/v1/transactions/${tx.processing_id}`))
        
        selectedTransaction.value = response.data
    } catch (error) {
        console.error('Error loading transaction details:', error)
        showNotification('Ошибка при загрузке деталей транзакции', 'error')
    }
}

const getStatusClass = (status) => {
    if (!status) return ''
    if (status === 'reviewed') return 'status-reviewed'
    if (status === 'pending_review') return 'status-pending'
    return 'status-default'
}

const getStatusText = (status) => {
    const statusMap = {
        'pending_review': 'Ожидает проверки',
        'reviewed': 'Проверено',
        'approved': 'Одобрено',
        'rejected': 'Отклонено'
    }
    return statusMap[status] || status
}

const getRiskClass = (riskLevel) => {
    if (!riskLevel) return ''
    if (riskLevel === 'high') return 'risk-high'
    if (riskLevel === 'medium') return 'risk-medium'
    if (riskLevel === 'low') return 'risk-low'
    return ''
}

const getRiskLevelText = (level) => {
    const levelMap = {
        'low': 'Низкий',
        'medium': 'Средний',
        'high': 'Высокий'
    }
    return levelMap[level] || level
}

const getFlagText = (flag) => {
    const flagMap = {
        'very_large_amount': 'Очень крупная сумма',
        'large_amount': 'Крупная сумма',
        'medium_amount': 'Средняя сумма',
        'offshore_counterparty': 'Офшорный контрагент',
        'unusual_time': 'Необычное время',
        'late_hours': 'Поздние часы',
        'high_frequency': 'Высокая частота',
        'medium_frequency': 'Средняя частота',
        'blacklisted_counterparty': 'Черный список',
        'international_transfer': 'Международный перевод',
        'withdrawal': 'Снятие средств',
        'large_atm_transaction': 'Крупная транзакция через банкомат',
        'atm_transaction': 'Транзакция через банкомат',
        'large_mobile_transaction': 'Крупная мобильная транзакция',
        'high_risk_currency': 'Высокорискованная валюта',
        'round_amount': 'Круглая сумма'
    }
    return flagMap[flag] || flag
}

const formatAmount = (amount, currency) => {
    if (amount === null || amount === undefined || amount === '') return 'N/A'
    const numAmount = typeof amount === 'string' ? parseFloat(amount) : amount
    if (isNaN(numAmount)) return 'N/A'
    return new Intl.NumberFormat('ru-RU', {
        style: 'currency',
        currency: currency || 'USD'
    }).format(numAmount)
}

const formatDate = (dateString) => {
    if (!dateString) return 'N/A'
    return new Date(dateString).toLocaleString('ru-RU')
}

const showNotification = (message, type = 'info') => {
    notification.value = { message, type }
    setTimeout(() => {
        notification.value = null
    }, 5000)
}

const loadLogs = async () => {
    try {
        const response = await axios.get('http://localhost:8080/api/v1/events?limit=100')
        events.value = response.data.events.reverse() // Новые сверху
    } catch (error) {
        console.error('Error loading logs:', error)
    }
}

const loadStats = async () => {
    try {
        const response = await axios.get('http://localhost:8080/api/v1/stats')
        stats.value = response.data
    } catch (error) {
        console.error('Error loading stats:', error)
    }
}

const refreshLogs = async () => {
    loading.value = true
    await loadLogs()
    await loadStats()
    loading.value = false
    showNotification('Логи обновлены', 'success')
}

const getComponentName = (component) => {
    const names = {
        'api': 'API',
        'sqlite': 'SQLite',
        'kafka': 'Kafka',
        'redis': 'Redis',
        'analyzer': 'Анализатор'
    }
    return names[component] || component
}

const getServiceName = (service) => {
    const names = {
        'ingestion-service': 'Сервис приема',
        'fraud-detection-service': 'Сервис детекции'
    }
    return names[service] || service
}

const getEventTypeName = (eventType) => {
    const names = {
        'transaction_received': 'Транзакция получена',
        'transaction_saved': 'Транзакция сохранена',
        'kafka_sent': 'Отправлено в Kafka',
        'kafka_received': 'Получено из Kafka',
        'redis_saved': 'Сохранено в Redis',
        'analysis_started': 'Анализ начат',
        'analysis_completed': 'Анализ завершен',
        'db_updated': 'БД обновлена'
    }
    return names[eventType] || eventType
}

const getLogClass = (event) => {
    return `log-${event.component}`
}

const getComponentClass = (component) => {
    return `component-${component}`
}

const formatTime = (timeString) => {
    if (!timeString) return ''
    const date = new Date(timeString)
    return date.toLocaleTimeString('ru-RU')
}

const formatLogValue = (value) => {
    if (Array.isArray(value)) {
        return value.join(', ')
    }
    if (typeof value === 'object') {
        return JSON.stringify(value)
    }
    return value
}

const generateRandomTransaction = async () => {
    loading.value = true
    try {
        const response = await axios.get('http://localhost:8080/api/v1/transactions/generate')
        
        // Заполняем форму сгенерированными данными
        form.value = {
            transaction_id: response.data.transaction_id,
            account_number: response.data.account_number,
            amount: response.data.amount,
            currency: response.data.currency,
            transaction_type: response.data.transaction_type,
            counterparty_account: response.data.counterparty_account || '',
            counterparty_bank: response.data.counterparty_bank || '',
            counterparty_country: response.data.counterparty_country || '',
            channel: response.data.channel,
            user_id: response.data.user_id || '',
            branch_id: response.data.branch_id || ''
        }

        showNotification('Форма заполнена случайными данными. Проверьте данные и нажмите "Отправить"', 'success')
    } catch (error) {
        showNotification('Ошибка при генерации транзакции: ' + (error.response?.data?.error || error.message), 'error')
    } finally {
        loading.value = false
    }
}

const clearDatabase = async () => {
    if (!confirm('Вы уверены, что хотите удалить ВСЕ транзакции из SQLite и Redis? Это действие нельзя отменить.')) {
        return
    }

    loading.value = true
    try {
        const response = await axios.delete('http://localhost:8080/api/v1/transactions')
        
        // Обновляем данные
        transactions.value = []
        selectedTransaction.value = null
        
        showNotification('Все транзакции и кэш успешно очищены', 'success')
        
        setTimeout(() => {
            loadLogs()
            loadStats()
        }, 1000)
    } catch (error) {
        showNotification('Ошибка при очистке БД: ' + (error.response?.data?.error || error.message), 'error')
    } finally {
        loading.value = false
    }
}

onMounted(() => {
    checkServices()
    loadTransactions()
    loadLogs()
    loadStats()
    setInterval(() => checkServices(), 5000)
    setInterval(() => loadTransactions(), 3000)
    setInterval(() => {
        loadLogs()
        loadStats()
    }, 2000)
})
</script>

