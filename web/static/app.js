// API Base URL
const API_BASE = '/api';

// State
let prices = [];
let appliances = [];
let recommendations = [];
let smartRecommendations = [];
let priceChart = null;

// UK DNO Region mapping (approximate boundaries)
const DNO_REGIONS = {
    'A': { name: 'Eastern England', lat: 52.5, lon: 0.5 },
    'B': { name: 'East Midlands', lat: 52.8, lon: -1.2 },
    'C': { name: 'London', lat: 51.5, lon: -0.1 },
    'D': { name: 'Merseyside and N Wales', lat: 53.4, lon: -3.0 },
    'E': { name: 'West Midlands', lat: 52.5, lon: -2.0 },
    'F': { name: 'North Eastern England', lat: 54.9, lon: -1.6 },
    'G': { name: 'North Western England', lat: 53.8, lon: -2.5 },
    'H': { name: 'Southern England', lat: 51.0, lon: -1.0 },
    'J': { name: 'South Eastern England', lat: 51.2, lon: 0.8 },
    'K': { name: 'South Wales', lat: 51.6, lon: -3.5 },
    'L': { name: 'South Western England', lat: 50.7, lon: -3.5 },
    'M': { name: 'Yorkshire', lat: 53.8, lon: -1.5 },
    'N': { name: 'Southern Scotland', lat: 55.8, lon: -3.5 },
    'P': { name: 'Northern Scotland', lat: 57.5, lon: -4.0 }
};

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    initTabs();
    initForms();
    loadStatus();
    loadHousehold();
    loadPrices();
    loadAppliances();
    loadRecommendations();
    loadSmartRecommendations();
    loadWeatherForecast();
});

// Tab Navigation
function initTabs() {
    const tabs = document.querySelectorAll('.tab');
    const tabContents = document.querySelectorAll('.tab-content');

    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const targetTab = tab.dataset.tab;

            // Update active tab
            tabs.forEach(t => t.classList.remove('active'));
            tab.classList.add('active');

            // Update active content
            tabContents.forEach(content => content.classList.remove('active'));
            document.getElementById(`${targetTab}-tab`).classList.add('active');
        });
    });
}

// Forms
function initForms() {
    // Add appliance modal
    document.getElementById('add-appliance-btn').addEventListener('click', () => {
        document.getElementById('add-appliance-form').style.display = 'flex';
    });

    document.getElementById('close-form').addEventListener('click', closeApplianceForm);
    document.getElementById('cancel-form').addEventListener('click', closeApplianceForm);

    document.getElementById('appliance-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        await addAppliance();
    });

    document.getElementById('settings-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        await saveSettings();
    });

    // Quiet hours toggle
    document.getElementById('enable-quiet-hours').addEventListener('change', (e) => {
        document.getElementById('quiet-hours-settings').style.display = e.target.checked ? 'block' : 'none';
    });
}

let editingApplianceId = null;

function closeApplianceForm() {
    document.getElementById('add-appliance-form').style.display = 'none';
    document.getElementById('appliance-form').reset();
    editingApplianceId = null;
    document.querySelector('#add-appliance-form h3').textContent = 'Add New Appliance';
    document.querySelector('#appliance-form button[type="submit"]').textContent = 'Add Appliance';
}

// API Calls
async function loadStatus() {
    try {
        const response = await fetch(`${API_BASE}/status`);
        const data = await response.json();
        document.getElementById('status-text').textContent = `Connected - Region ${data.region}`;
    } catch (error) {
        document.getElementById('status-text').textContent = 'Disconnected';
        console.error('Failed to load status:', error);
    }
}

async function loadHousehold() {
    try {
        const response = await fetch(`${API_BASE}/household`);
        const household = await response.json();

        // Populate settings form
        if (household) {
            document.getElementById('household-name').value = household.Name || 'My Household';
            document.getElementById('household-region').value = household.Region || 'C';
            document.getElementById('household-lat').value = household.Latitude || '';
            document.getElementById('household-lon').value = household.Longitude || '';
            document.getElementById('stagger-loads').checked = household.StaggerHeavyLoads || false;

            if (household.QuietHours && household.QuietHours.length > 0) {
                document.getElementById('quiet-start').value = household.QuietHours[0].Start || '22:00';
                document.getElementById('quiet-end').value = household.QuietHours[0].End || '07:00';
            }
        }
    } catch (error) {
        console.error('Failed to load household:', error);
    }
}

async function loadPrices() {
    try {
        const response = await fetch(`${API_BASE}/prices`);
        prices = await response.json();
        renderPriceChart();
        renderPricesTable();
    } catch (error) {
        console.error('Failed to load prices:', error);
    }
}

async function loadAppliances() {
    try {
        const response = await fetch(`${API_BASE}/appliances`);
        appliances = await response.json();
        renderAppliances();
    } catch (error) {
        console.error('Failed to load appliances:', error);
    }
}

async function loadRecommendations() {
    const container = document.getElementById('recommendations');
    const loading = document.getElementById('recommendations-loading');

    try {
        loading.style.display = 'block';
        container.innerHTML = '';

        const response = await fetch(`${API_BASE}/recommendations`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({})
        });

        recommendations = await response.json();
        loading.style.display = 'none';
        renderRecommendations();
    } catch (error) {
        loading.style.display = 'none';
        console.error('Failed to load recommendations:', error);
        container.innerHTML = `
            <div class="empty-state">
                <p>Failed to load recommendations. Make sure you have appliances configured.</p>
            </div>
        `;
    }
}

async function addAppliance() {
    const appliance = {
        Name: document.getElementById('appliance-name').value,
        CycleMinutes: parseInt(document.getElementById('appliance-cycle').value),
        EstKWh: parseFloat(document.getElementById('appliance-kwh').value),
        NoiseLevel: parseInt(document.getElementById('appliance-noise').value),
        Priority: parseInt(document.getElementById('appliance-priority').value),
        ControlType: document.getElementById('appliance-control').value,
        UsageFrequency: document.getElementById('appliance-frequency').value,
        Class: document.getElementById('appliance-class').value,
        CoupledApplianceID: document.getElementById('appliance-coupled').value,
        CanWaitDays: parseInt(document.getElementById('appliance-can-wait').value),
        Enabled: true,
        AllowedWindows: [],
        BlockedWindows: []
    };

    try {
        let response;
        if (editingApplianceId) {
            // Update existing appliance
            response = await fetch(`${API_BASE}/appliances/${editingApplianceId}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(appliance)
            });
        } else {
            // Create new appliance
            response = await fetch(`${API_BASE}/appliances`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(appliance)
            });
        }

        if (response.ok) {
            closeApplianceForm();
            await loadAppliances();
            await loadRecommendations();
            await loadSmartRecommendations();
        }
    } catch (error) {
        console.error('Failed to save appliance:', error);
        alert('Failed to save appliance');
    }
}

async function editAppliance(id) {
    try {
        const response = await fetch(`${API_BASE}/appliances/${id}`);
        const appliance = await response.json();

        // Populate form
        document.getElementById('appliance-name').value = appliance.Name;
        document.getElementById('appliance-cycle').value = appliance.CycleMinutes;
        document.getElementById('appliance-kwh').value = appliance.EstKWh;
        document.getElementById('appliance-noise').value = appliance.NoiseLevel;
        document.getElementById('appliance-priority').value = appliance.Priority;
        document.getElementById('appliance-control').value = appliance.ControlType || 'manual';
        document.getElementById('appliance-frequency').value = appliance.UsageFrequency || 'on_demand';
        document.getElementById('appliance-class').value = appliance.Class || 'standalone';
        document.getElementById('appliance-coupled').value = appliance.CoupledApplianceID || '';
        document.getElementById('appliance-can-wait').value = appliance.CanWaitDays || 0;
        updateWaitDaysLabel(appliance.CanWaitDays || 0);
        toggleCoupledFields();

        // Update form UI
        editingApplianceId = id;
        document.querySelector('#add-appliance-form h3').textContent = 'Edit Appliance';
        document.querySelector('#appliance-form button[type="submit"]').textContent = 'Save Changes';
        document.getElementById('add-appliance-form').style.display = 'flex';
    } catch (error) {
        console.error('Failed to load appliance:', error);
        alert('Failed to load appliance');
    }
}

async function deleteAppliance(id) {
    if (!confirm('Are you sure you want to delete this appliance?')) {
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/appliances/${id}`, {
            method: 'DELETE'
        });

        if (response.ok) {
            await loadAppliances();
            await loadRecommendations();
        }
    } catch (error) {
        console.error('Failed to delete appliance:', error);
        alert('Failed to delete appliance');
    }
}

function detectRegion(lat, lon) {
    let closestRegion = 'C'; // Default to London
    let minDistance = Infinity;

    for (const [code, region] of Object.entries(DNO_REGIONS)) {
        const distance = Math.sqrt(
            Math.pow(lat - region.lat, 2) + Math.pow(lon - region.lon, 2)
        );
        if (distance < minDistance) {
            minDistance = distance;
            closestRegion = code;
        }
    }

    return closestRegion;
}

function autoDetectRegion() {
    console.log('autoDetectRegion called');

    if (!navigator.geolocation) {
        alert('Geolocation is not supported by your browser');
        return;
    }

    const button = document.getElementById('auto-detect-region');
    console.log('Button found:', button);

    button.textContent = 'Detecting...';
    button.disabled = true;

    console.log('Requesting geolocation...');
    navigator.geolocation.getCurrentPosition(
        (position) => {
            console.log('Position received:', position);
            const { latitude, longitude } = position.coords;
            const region = detectRegion(latitude, longitude);
            document.getElementById('household-region').value = region;
            document.getElementById('household-lat').value = latitude;
            document.getElementById('household-lon').value = longitude;
            button.textContent = '📍 Auto-detect Region';
            button.disabled = false;
            alert(`Detected region: ${region} - ${DNO_REGIONS[region].name}\nLocation updated! Click "Save Settings" to apply.`);
        },
        (error) => {
            console.error('Geolocation error:', error);
            button.textContent = '📍 Auto-detect Region';
            button.disabled = false;
            alert(`Unable to detect location: ${error.message}`);
        }
    );
}

// Make function available globally for onclick
window.autoDetectRegion = autoDetectRegion;

async function saveSettings() {
    const household = {
        Name: document.getElementById('household-name').value,
        Region: document.getElementById('household-region').value,
        Latitude: parseFloat(document.getElementById('household-lat').value) || 0,
        Longitude: parseFloat(document.getElementById('household-lon').value) || 0,
        StaggerHeavyLoads: document.getElementById('stagger-loads').checked,
        QuietHours: [{
            Start: document.getElementById('quiet-start').value,
            End: document.getElementById('quiet-end').value,
            DaysOfWeek: [1, 2, 3, 4, 5, 6, 7]
        }],
        BlockedWindows: [],
        CarbonWeight: 0.0
    };

    try {
        const response = await fetch(`${API_BASE}/household`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(household)
        });

        if (response.ok) {
            alert('Settings saved successfully!');
            await loadRecommendations();
        }
    } catch (error) {
        console.error('Failed to save settings:', error);
        alert('Failed to save settings');
    }
}

// Render Functions
function renderRecommendations() {
    const container = document.getElementById('recommendations');

    if (!recommendations || recommendations.length === 0) {
        const hasAppliances = appliances && appliances.length > 0;
        const message = hasAppliances
            ? 'No time slots available tonight. Check back tomorrow morning for recommendations!'
            : 'Add some appliances to get started!';

        container.innerHTML = `
            <div class="empty-state">
                <p>${message}</p>
            </div>
        `;
        return;
    }

    // Split recommendations into today and tomorrow
    const now = new Date();
    const todayEnd = new Date(now);
    todayEnd.setHours(23, 59, 59, 999);

    const todayRecs = [];
    const tomorrowRecs = [];

    recommendations.forEach(rec => {
        const todayWindows = [];
        const tomorrowWindows = [];

        rec.recommendations.forEach(w => {
            const startTime = new Date(w.Start);
            if (startTime <= todayEnd) {
                todayWindows.push(w);
            } else {
                tomorrowWindows.push(w);
            }
        });

        if (todayWindows.length > 0) {
            todayRecs.push({ appliance: rec.appliance, recommendations: todayWindows });
        }
        if (tomorrowWindows.length > 0) {
            tomorrowRecs.push({ appliance: rec.appliance, recommendations: tomorrowWindows });
        }
    });

    let html = '';

    if (todayRecs.length > 0) {
        html += '<h2 style="margin-bottom: 1rem; color: var(--primary);">Today</h2>';
        html += todayRecs.map((rec, index) => renderRecommendationCard(rec, index)).join('');
    }

    if (tomorrowRecs.length > 0) {
        html += '<h2 style="margin-top: 2rem; margin-bottom: 1rem; color: var(--secondary);">Tomorrow</h2>';
        html += tomorrowRecs.map((rec, index) => renderRecommendationCard(rec, index)).join('');
    }

    container.innerHTML = html;
}

function renderRecommendationCard(rec, index) {
    const cost = rec.recommendations[0].CostGBP;
    const costStr = cost < 0 ? `+£${Math.abs(cost).toFixed(2)}` : `£${cost.toFixed(2)}`;

    // Format time slots
    let timeDisplay;
    if (rec.recommendations.length === 1) {
        timeDisplay = `${formatTime(rec.recommendations[0].Start)} - ${formatTime(rec.recommendations[0].End)}`;
    } else {
        timeDisplay = rec.recommendations.map(w =>
            `${formatTime(w.Start)} - ${formatTime(w.End)}`
        ).join(' or ');
    }

    return `
        <div class="recommendation-card">
            <div class="rec-header">
                <div class="rec-appliance">Best time to run ${rec.appliance}</div>
            </div>
            <div class="rec-best-time">
                <div class="best-time-slots">${timeDisplay}</div>
                <div class="best-time-cost ${cost < 0 ? 'negative' : 'low'}">${costStr}</div>
            </div>
        </div>
    `;
}

function renderAppliances() {
    const container = document.getElementById('appliances-list');

    if (!appliances || appliances.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <p>No appliances configured yet.</p>
                <p>Click "Add Appliance" to get started!</p>
            </div>
        `;
        return;
    }

    container.innerHTML = appliances.map(app => `
        <div class="appliance-card">
            <div class="appliance-info">
                <h3>${app.Name}</h3>
                <div class="appliance-meta">
                    <span>⏱️ ${app.CycleMinutes} min</span>
                    <span>⚡ ${app.EstKWh} kWh</span>
                    <span>🔊 Noise: ${app.NoiseLevel}/5</span>
                    <span>⭐ Priority: ${app.Priority}/5</span>
                </div>
                <div class="appliance-meta">
                    <span>🎮 ${app.ControlType === 'smart' ? 'Smart' : 'Manual'}</span>
                    <span>📅 ${formatFrequency(app.UsageFrequency)}</span>
                </div>
            </div>
            <div class="appliance-actions">
                <button class="btn btn-secondary btn-sm" onclick="editAppliance('${app.ID}')">Edit</button>
                <button class="btn btn-danger btn-sm" onclick="deleteAppliance('${app.ID}')">Delete</button>
            </div>
        </div>
    `).join('');
}

function formatFrequency(freq) {
    const map = {
        'daily': 'Daily',
        '3x_week': '3x/week',
        'weekly': 'Weekly',
        'on_demand': 'On-demand'
    };
    return map[freq] || freq;
}

function renderPriceChart() {
    if (!prices || prices.length === 0) return;

    const ctx = document.getElementById('priceChart').getContext('2d');

    // Destroy existing chart
    if (priceChart) {
        priceChart.destroy();
    }

    // Filter to current and future prices (include slot we're currently in)
    const now = new Date();
    const futurePrices = prices.filter(p => new Date(p.End) > now);

    const labels = futurePrices.map(p => formatTime(p.Start));
    const data = futurePrices.map(p => p.PencePerKWh);

    priceChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: labels,
            datasets: [{
                label: 'Price (p/kWh)',
                data: data,
                borderColor: 'rgb(37, 99, 235)',
                backgroundColor: 'rgba(37, 99, 235, 0.1)',
                fill: true,
                tension: 0.4
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    display: false
                },
                tooltip: {
                    callbacks: {
                        label: (context) => {
                            return `${context.parsed.y.toFixed(2)} p/kWh`;
                        }
                    }
                }
            },
            scales: {
                y: {
                    beginAtZero: false,
                    ticks: {
                        callback: (value) => value.toFixed(1) + 'p'
                    }
                },
                x: {
                    ticks: {
                        maxTicksLimit: 12
                    }
                }
            }
        }
    });
}

function renderPricesTable() {
    const container = document.getElementById('prices-table');

    if (!prices || prices.length === 0) {
        container.innerHTML = '<div class="empty-state">No price data available</div>';
        return;
    }

    const sorted = [...prices].sort((a, b) => a.PencePerKWh - b.PencePerKWh);
    const min = sorted[0].PencePerKWh;
    const max = sorted[sorted.length - 1].PencePerKWh;

    container.innerHTML = prices.map(p => {
        let priceClass = '';
        if (p.PencePerKWh < 0) priceClass = 'negative';
        else if (p.PencePerKWh < min + (max - min) * 0.3) priceClass = 'low';
        else if (p.PencePerKWh > max - (max - min) * 0.3) priceClass = 'high';

        return `
            <div class="price-row">
                <span class="price-time">${formatDateTime(p.Start)}</span>
                <span class="price-value ${priceClass}">
                    ${p.PencePerKWh.toFixed(2)} p/kWh
                </span>
            </div>
        `;
    }).join('');
}

// Utility Functions
function formatTime(dateStr) {
    const date = new Date(dateStr);
    return date.toLocaleTimeString('en-GB', {
        hour: '2-digit',
        minute: '2-digit',
        hour12: false
    });
}

function formatDateTime(dateStr) {
    const date = new Date(dateStr);
    const today = new Date();
    const tomorrow = new Date(today);
    tomorrow.setDate(tomorrow.getDate() + 1);

    let dayPrefix = '';
    if (date.toDateString() === today.toDateString()) {
        dayPrefix = 'Today ';
    } else if (date.toDateString() === tomorrow.toDateString()) {
        dayPrefix = 'Tomorrow ';
    }

    return dayPrefix + date.toLocaleTimeString('en-GB', {
        hour: '2-digit',
        minute: '2-digit',
        hour12: false
    });
}

// Smart Recommendations
async function loadSmartRecommendations() {
    try {
        const response = await fetch(`${API_BASE}/smart-recommendations`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({})
        });

        if (response.ok) {
            smartRecommendations = await response.json();
            renderSmartRecommendations();
        }
    } catch (error) {
        console.error('Failed to load smart recommendations:', error);
    }
}

function renderSmartRecommendations() {
    const container = document.getElementById('smart-recommendations');
    const section = document.getElementById('smart-recommendations-section');

    if (!smartRecommendations || smartRecommendations.length === 0) {
        section.style.display = 'none';
        return;
    }

    section.style.display = 'block';

    container.innerHTML = smartRecommendations.map(smart => {
        const bestOption = smart.Options[smart.BestOptionIndex];
        const otherOptions = smart.Options.filter((_, i) => i !== smart.BestOptionIndex);

        return `
            <div class="smart-rec-card">
                <h3>${smart.ApplianceName}</h3>
                <div class="best-option">
                    <div class="option-badge">✨ Recommended</div>
                    <div class="option-content">
                        <div class="option-day">${bestOption.Day}</div>
                        ${bestOption.Weather ? renderWeather(bestOption.Weather) : ''}
                        <div class="option-cost">
                            ${bestOption.UsesNaturalDry ? '☀️ Wash & line dry' : '🔥 Wash & tumble dry'}
                            <strong>£${bestOption.TotalCostGBP.toFixed(2)}</strong>
                            ${bestOption.SavingsVsToday > 0 ? `<span class="savings">Save £${bestOption.SavingsVsToday.toFixed(2)}!</span>` : ''}
                        </div>
                        <div class="option-recommendation">${bestOption.Recommendation}</div>
                    </div>
                </div>
                ${otherOptions.length > 0 ? `
                    <details class="other-options">
                        <summary>Show ${otherOptions.length} other option${otherOptions.length > 1 ? 's' : ''}</summary>
                        ${otherOptions.map(opt => `
                            <div class="option-content alt-option">
                                <div class="option-day">${opt.Day}</div>
                                ${opt.Weather ? renderWeather(opt.Weather) : ''}
                                <div class="option-cost">
                                    ${opt.UsesNaturalDry ? '☀️ Wash & line dry' : '🔥 Wash & tumble dry'}
                                    £${opt.TotalCostGBP.toFixed(2)}
                                </div>
                                <div class="option-recommendation">${opt.Recommendation}</div>
                            </div>
                        `).join('')}
                    </details>
                ` : ''}
            </div>
        `;
    }).join('');
}

function renderWeather(weather) {
    if (!weather) return '';
    return `
        <div class="weather-info">
            ${weather.IsSunny ? '☀️' : '☁️'}
            ${weather.MaxTempC.toFixed(0)}°C,
            ${weather.SunshineHours.toFixed(1)}h sun,
            ${weather.PrecipProb.toFixed(0)}% rain
        </div>
    `;
}

async function loadWeatherForecast() {
    try {
        const response = await fetch(`${API_BASE}/weather`);

        if (response.ok) {
            const weatherDays = await response.json();
            if (weatherDays && weatherDays.length > 0) {
                renderWeatherWidget(weatherDays);
            }
        }
    } catch (error) {
        console.error('Failed to load weather:', error);
    }
}

function renderWeatherWidget(weatherDays) {
    const widget = document.getElementById('weather-widget');
    const display = document.getElementById('weather-display');

    if (!weatherDays || weatherDays.length === 0) {
        widget.style.display = 'none';
        return;
    }

    widget.style.display = 'block';

    const now = new Date();
    // Use local date for comparison (not UTC)
    const today = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
    const tomorrowDate = new Date(now);
    tomorrowDate.setDate(tomorrowDate.getDate() + 1);
    const tomorrow = `${tomorrowDate.getFullYear()}-${String(tomorrowDate.getMonth() + 1).padStart(2, '0')}-${String(tomorrowDate.getDate()).padStart(2, '0')}`;

    display.innerHTML = weatherDays.map((w, idx) => {
        const weatherDate = w.Date.split('T')[0];
        let dayName;
        if (weatherDate === today) {
            dayName = 'Today';
        } else if (weatherDate === tomorrow) {
            dayName = 'Tomorrow';
        } else {
            const date = new Date(weatherDate + 'T12:00:00');  // Use date part only
            dayName = date.toLocaleDateString('en-GB', { weekday: 'long' });
        }

        const isDryingWeather = w.IsSunny;

        return `
            <div style="display: flex; align-items: center; gap: 15px; padding: 12px; ${idx > 0 ? 'border-top: 1px solid rgba(255,255,255,0.1);' : ''}">
                <div style="min-width: 100px; font-weight: 600;">${dayName}</div>
                <div style="font-size: 36px;">${w.IsSunny ? '☀️' : '☁️'}</div>
                <div style="flex: 1;">
                    <div style="font-size: 20px; font-weight: bold;">
                        ${w.MinTempC.toFixed(0)}°C - ${w.MaxTempC.toFixed(0)}°C
                    </div>
                    <div style="font-size: 14px; opacity: 0.9;">
                        ${w.SunshineHours.toFixed(1)}h sun, ${w.PrecipProb.toFixed(0)}% rain
                    </div>
                </div>
                ${isDryingWeather ? '<div style="color: #4CAF50; font-weight: 600; min-width: 120px;">☀️ Good for drying</div>' : '<div style="color: #FF9800; font-weight: 600; min-width: 120px;">☁️ Use dryer</div>'}
            </div>
        `;
    }).join('');
}

// Form helpers
function toggleCoupledFields() {
    const classSelect = document.getElementById('appliance-class');
    const coupledGroup = document.getElementById('coupled-appliance-group');
    const canWaitGroup = document.getElementById('can-wait-group');

    if (classSelect.value === 'coupled') {
        coupledGroup.style.display = 'block';
        canWaitGroup.style.display = 'block';
        updateCoupledApplianceDropdown();
    } else {
        coupledGroup.style.display = 'none';
        canWaitGroup.style.display = 'none';
    }
}

function updateCoupledApplianceDropdown() {
    const dropdown = document.getElementById('appliance-coupled');
    dropdown.innerHTML = '<option value="">None</option>';

    appliances.forEach(app => {
        if (app.Class === 'weather_dependent' || app.Class === 'standalone') {
            dropdown.innerHTML += `<option value="${app.ID}">${app.Name}</option>`;
        }
    });
}

function updateWaitDaysLabel(value) {
    document.getElementById('wait-days-label').textContent = `${value} day${value == 1 ? '' : 's'}`;
}

window.toggleCoupledFields = toggleCoupledFields;
window.updateWaitDaysLabel = updateWaitDaysLabel;
