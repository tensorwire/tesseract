import './style.css';
import {GetStatus, ListModels, LoadModel, SendMessage, RunCommand, PullModelWithProgress, CancelPull} from '../wailsjs/go/main/App';

const messagesEl = document.getElementById('messages');
const inputEl = document.getElementById('input');
const sendBtn = document.getElementById('send');
const statusDot = document.getElementById('status-dot');
const statusText = document.getElementById('status-text');
const modelSelect = document.getElementById('model-select');
const cmdPalette = document.getElementById('cmd-palette');
const hintEl = document.getElementById('hint');

let sending = false;
let paletteIndex = -1;

const commands = [
    { cmd: '/help', desc: 'Show all commands' },
    { cmd: '/pull', desc: 'Download model from HuggingFace', arg: '<org/model>' },
    { cmd: '/models', desc: 'List downloaded models' },
    { cmd: '/load', desc: 'Load model into server', arg: '<model>' },
    { cmd: '/info', desc: 'Show model architecture', arg: '<model>' },
    { cmd: '/status', desc: 'Show server status' },
    { cmd: '/gpus', desc: 'Detect hardware' },
    { cmd: '/bench', desc: 'GPU benchmark' },
    { cmd: '/quantize', desc: 'Quantize model', arg: '<model> [q8|q4]' },
    { cmd: '/train', desc: 'Train a model', arg: 'data=<file>' },
    { cmd: '/stop', desc: 'Stop server daemon' },
];

function showPalette(filter) {
    const matches = commands.filter(c =>
        c.cmd.startsWith(filter) || filter === '/'
    );
    if (matches.length === 0) {
        hidePalette();
        return;
    }
    paletteIndex = -1;
    cmdPalette.innerHTML = matches.map((c, i) => {
        const arg = c.arg ? ` <span class="cmd-arg">${c.arg}</span>` : '';
        return `<div class="cmd-item" data-cmd="${c.cmd}" data-index="${i}">
            <span class="cmd-name">${c.cmd}</span>${arg}
            <span class="cmd-desc">${c.desc}</span>
        </div>`;
    }).join('');
    cmdPalette.style.display = 'block';
    hintEl.style.display = 'none';

    cmdPalette.querySelectorAll('.cmd-item').forEach(item => {
        item.addEventListener('click', () => {
            const cmd = item.dataset.cmd;
            inputEl.value = cmd + ' ';
            inputEl.focus();
            hidePalette();
        });
    });
}

function hidePalette() {
    cmdPalette.style.display = 'none';
    cmdPalette.innerHTML = '';
    paletteIndex = -1;
}

function navigatePalette(dir) {
    const items = cmdPalette.querySelectorAll('.cmd-item');
    if (items.length === 0) return;
    items.forEach(i => i.classList.remove('active'));
    paletteIndex += dir;
    if (paletteIndex < 0) paletteIndex = items.length - 1;
    if (paletteIndex >= items.length) paletteIndex = 0;
    items[paletteIndex].classList.add('active');
    items[paletteIndex].scrollIntoView({ block: 'nearest' });
}

function selectPaletteItem() {
    const items = cmdPalette.querySelectorAll('.cmd-item');
    if (paletteIndex >= 0 && paletteIndex < items.length) {
        const cmd = items[paletteIndex].dataset.cmd;
        inputEl.value = cmd + ' ';
        inputEl.focus();
        hidePalette();
        return true;
    }
    return false;
}

inputEl.addEventListener('input', () => {
    inputEl.style.height = 'auto';
    inputEl.style.height = Math.min(inputEl.scrollHeight, 120) + 'px';

    const val = inputEl.value;
    if (val.startsWith('/') && !val.includes('\n')) {
        showPalette(val.split(' ')[0]);
    } else {
        hidePalette();
    }

    if (val.length === 0) {
        hintEl.style.display = '';
    } else {
        hintEl.style.display = 'none';
    }
});

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
    const paletteVisible = cmdPalette.style.display === 'block';

    if (paletteVisible && (e.key === 'ArrowDown' || e.key === 'ArrowUp')) {
        e.preventDefault();
        navigatePalette(e.key === 'ArrowDown' ? 1 : -1);
        return;
    }
    if (paletteVisible && (e.key === 'Tab' || (e.key === 'Enter' && paletteIndex >= 0))) {
        e.preventDefault();
        selectPaletteItem();
        return;
    }
    if (e.key === 'Escape') {
        hidePalette();
        return;
    }
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        hidePalette();
        send();
    }
});

inputEl.addEventListener('blur', () => {
    setTimeout(hidePalette, 150);
});

refreshStatus();
refreshModels();
setInterval(refreshStatus, 10000);
