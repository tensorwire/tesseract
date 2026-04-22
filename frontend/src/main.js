import './style.css';
import {GetStatus, ListModels, LoadModel, SendMessage, RunCommand, PullModelWithProgress, CancelPull} from '../wailsjs/go/main/App';

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
        const [models, status] = await Promise.all([ListModels(), GetStatus()]);
        modelSelect.innerHTML = '<option value="">select model</option>';
        if (models) {
            for (const m of models) {
                const opt = document.createElement('option');
                opt.value = m;
                opt.textContent = m;
                if (status.model && m === status.model) {
                    opt.selected = true;
                }
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

    // Slash commands
    if (text.startsWith('/')) {
        addMessage('user', text);
        const cmdText = text.slice(1);

        // Pull gets a progress indicator with cancel
        if (cmdText.startsWith('pull ')) {
            const modelName = cmdText.slice(5).trim();
            const progressEl = addMessage('system', '');
            const bar = document.createElement('div');
            bar.className = 'pull-progress';
            bar.innerHTML = `
                <span class="pull-text">downloading ${modelName}...</span>
                <button class="pull-cancel" title="Cancel">✕</button>
            `;
            progressEl.textContent = '';
            progressEl.appendChild(bar);
            bar.querySelector('.pull-cancel').addEventListener('click', async () => {
                const result = await CancelPull();
                progressEl.textContent = result;
            });
            try {
                const result = await PullModelWithProgress(modelName);
                progressEl.textContent = result;
                await refreshModels();
                await refreshStatus();
            } catch (err) {
                progressEl.textContent = `[error: ${err}]`;
            }
        } else {
            const resultEl = addMessage('system', 'running...');
            try {
                const result = await RunCommand(cmdText);
                resultEl.textContent = result;
                if (cmdText === 'models' || cmdText.startsWith('load ')) {
                    await refreshModels();
                    await refreshStatus();
                }
            } catch (err) {
                resultEl.textContent = `[error: ${err}]`;
            }
        }

        sending = false;
        sendBtn.disabled = false;
        inputEl.focus();
        return;
    }

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
