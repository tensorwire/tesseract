import './style.css';
import {GetStatus, ListModels, LoadModel, SendMessage} from '../wailsjs/go/main/App';

const messagesEl = document.getElementById('messages');
const inputEl = document.getElementById('input');
const sendBtn = document.getElementById('send');
const statusDot = document.getElementById('status-dot');
const statusText = document.getElementById('status-text');
const modelSelect = document.getElementById('model-select');

let sending = false;

async function refreshStatus() {
    try {
        const status = await GetStatus();
        if (status.running) {
            statusDot.className = 'connected';
            let text = status.model || 'no model';
            if (status.gpu) text += ' (GPU)';
            statusText.textContent = text;
        } else {
            statusDot.className = 'error';
            statusText.textContent = 'disconnected';
        }
    } catch {
        statusDot.className = 'error';
        statusText.textContent = 'disconnected';
    }
}

async function refreshModels() {
    try {
        const models = await ListModels();
        modelSelect.innerHTML = '<option value="">select model</option>';
        if (models) {
            for (const m of models) {
                const opt = document.createElement('option');
                opt.value = m;
                opt.textContent = m;
                modelSelect.appendChild(opt);
            }
        }
    } catch {}
}

modelSelect.addEventListener('change', async () => {
    const model = modelSelect.value;
    if (!model) return;
    statusText.textContent = `loading ${model}...`;
    statusDot.className = '';
    try {
        await LoadModel(model);
        await refreshStatus();
    } catch (err) {
        statusText.textContent = `error: ${err}`;
        statusDot.className = 'error';
    }
});

function addMessage(role, content) {
    const div = document.createElement('div');
    div.className = `message ${role}`;
    div.textContent = content;
    messagesEl.appendChild(div);
    messagesEl.scrollTop = messagesEl.scrollHeight;
    return div;
}

async function send() {
    const text = inputEl.value.trim();
    if (!text || sending) return;

    sending = true;
    sendBtn.disabled = true;
    inputEl.value = '';
    inputEl.style.height = 'auto';

    addMessage('user', text);
    const thinkingEl = addMessage('assistant', 'thinking...');
    thinkingEl.classList.add('thinking');

    try {
        const response = await SendMessage(text);
        thinkingEl.textContent = response;
        thinkingEl.classList.remove('thinking');
    } catch (err) {
        thinkingEl.textContent = `[error: ${err}]`;
        thinkingEl.classList.remove('thinking');
    }

    sending = false;
    sendBtn.disabled = false;
    inputEl.focus();
}

sendBtn.addEventListener('click', send);
inputEl.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        send();
    }
});

inputEl.addEventListener('input', () => {
    inputEl.style.height = 'auto';
    inputEl.style.height = Math.min(inputEl.scrollHeight, 120) + 'px';
});

refreshStatus();
refreshModels();
setInterval(refreshStatus, 10000);
