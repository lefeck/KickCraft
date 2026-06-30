// ISO Generator Module
// Handles ISO generation and download

class ISOGenerator {
    constructor() {
        this.currentTask = null;
        this.pollInterval = null;
        this.logOffset = 0;
    }

    async generate() {
        console.log('[ISOGenerator.generate] called');
        const config = window.getConfigFromForm();

        const sourceType = document.querySelector('input[name="isoSourceType"]:checked')?.value;
        console.log('[ISOGenerator.generate] sourceType:', sourceType);

        let distro = null;
        if (sourceType === 'download') {
            distro = document.getElementById('distroDownloadSelect')?.value;
        } else {
            distro = document.getElementById('sourceISO')?.value;
        }
        console.log('[ISOGenerator.generate] distro:', distro);

        if (!distro) {
            const errorMsg = sourceType === 'download'
                ? 'Please select a distribution'
                : 'Please upload a local ISO file first';
            console.warn('[ISOGenerator.generate] no distro:', errorMsg);
            window.showToast(errorMsg, 'error');
            return;
        }

        // Front-end required-field validation (matches UbuntuCraft pattern).
        // Blocks the build if any asterisk-marked fields are empty.
        if (window.validateBasicRequired && !window.validateBasicRequired('isoStatus')) {
            console.warn('[ISOGenerator.generate] basic required fields missing, aborting');
            return;
        }

        // ISO Builder specific required fields: Destination ISO Name and
        // Kickstart Configuration. These are only validated here at build
        // time, not by the Configuration page buttons (Load Default /
        // Validate / Preview), which should remain usable while the
        // user is still authoring their config.
        const isoBuilderErrors = [];
        const destinationISOEl = document.getElementById('destinationISO');
        const ksConfigContentEl = document.getElementById('ksConfigContent');
        if (destinationISOEl) destinationISOEl.classList.remove('input-error');
        if (ksConfigContentEl) ksConfigContentEl.classList.remove('input-error');

        if (!destinationISOEl || !destinationISOEl.value || destinationISOEl.value.trim() === '') {
            if (destinationISOEl) destinationISOEl.classList.add('input-error');
            isoBuilderErrors.push('Destination ISO Name is required');
        }
        if (!ksConfigContentEl || !ksConfigContentEl.value || ksConfigContentEl.value.trim() === '') {
            if (ksConfigContentEl) ksConfigContentEl.classList.add('input-error');
            isoBuilderErrors.push('Kickstart Configuration is required');
        }
        if (isoBuilderErrors.length > 0) {
            // Scroll to first missing field
            const firstMissing = [destinationISOEl, ksConfigContentEl].find(
                el => el && (!el.value || el.value.trim() === '')
            );
            if (firstMissing && typeof firstMissing.scrollIntoView === 'function') {
                firstMissing.scrollIntoView({ behavior: 'smooth', block: 'center' });
            }
            // Show toast notification only (no inline status block)
            window.showToast(isoBuilderErrors.join(', '), 'error');
            return;
        }

        this.showProgress();

        try {
            await this.validate(config);
            console.log('[ISOGenerator.generate] validation passed');

            const destinationISO = document.getElementById('destinationISO')?.value || '';
            const installMedia = document.querySelector('input[name="installMediaType"]:checked')?.value || 'cdrom';
            const ethNaming = document.getElementById('ethNamingCheckbox')?.checked || false;
            const serialConsole = document.getElementById('serialConsoleCheckbox')?.checked || false;
            const baudRate = document.getElementById('baudRateSelect')?.value || '115200';
            const requestBody = {
                sourceType: sourceType,
                destinationISO: destinationISO,
                installMedia: installMedia,
                ethNaming: ethNaming,
                serialConsole: serialConsole,
                baudRate: baudRate,
                config: config
            };
            if (sourceType === 'download') {
                requestBody.distro = distro;
            } else {
                requestBody.sourceIso = distro;
            }
            console.log('[ISOGenerator.generate] request body:', requestBody);
            const response = await fetch('/api/iso/generate', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(requestBody)
            });
            console.log('[ISOGenerator.generate] response status:', response.status);

            const data = await response.json();
            console.log('[ISOGenerator.generate] response data:', data);

            if (!data.success) {
                throw new Error(data.error || 'Failed to start generation');
            }

            this.currentTask = data.data.taskId;
            console.log('[ISOGenerator.generate] currentTask set to:', this.currentTask);
            this.startPolling();

        } catch (error) {
            console.error('[ISOGenerator.generate] caught error:', error);
            this.updateProgressError(error.message);
            window.showToast(error.message, 'error');
        }
    }

    async validate(config) {
        console.log('[ISOGenerator.validate] sending config to /api/config/validate');
        const response = await fetch('/api/config/validate', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(config)
        });

        const data = await response.json();
        console.log('[ISOGenerator.validate] response:', data);

        if (data.success && data.data && data.data.data) {
            const validation = data.data.data;
            if (!validation.valid && validation.errors && validation.errors.length > 0) {
                throw new Error(`Validation failed: ${validation.errors.join(', ')}`);
            }
        }
    }

    startPolling() {
        if (this.pollInterval) {
            clearInterval(this.pollInterval);
        }

        this.pollInterval = setInterval(() => {
            this.checkStatus();
        }, 2000);
    }

    stopPolling() {
        if (this.pollInterval) {
            clearInterval(this.pollInterval);
            this.pollInterval = null;
        }
    }

    async checkStatus() {
        if (!this.currentTask) return;

        try {
            const url = `/api/iso/status/${this.currentTask}?logOffset=${this.logOffset}`;
            const response = await fetch(url);
            const data = await response.json();

            if (data.success && data.data) {
                const task = data.data;
                console.log('[ISOGenerator.checkStatus] task:', task);

                this.updateProgressFromStatus(task);

                if (task.status === 'completed') {
                    this.stopPolling();
                    this.onComplete(task);
                } else if (task.status === 'failed') {
                    this.stopPolling();
                    this.updateProgressError(task.error || 'Generation failed');
                    window.showToast('ISO generation failed', 'error');
                }
            }
        } catch (error) {
            console.error('Status check failed:', error);
        }
    }

    updateProgressFromStatus(task) {
        // Update progress bar
        const fill = document.getElementById('progressFill');
        if (fill) {
            fill.style.width = `${task.progress || 0}%`;
        }

        // Update steps visual state
        if (task.steps) {
            this.updateSteps(task.steps);
        }

        // Update message
        if (task.message) {
            const text = document.getElementById('progressText');
            if (text) {
                text.textContent = task.message;
            }
        }

        // Append new logs
        if (task.logs && task.logs.length > 0) {
            const log = document.getElementById('logContainer');
            if (log) {
                task.logs.forEach(msg => {
                    const entry = document.createElement('div');
                    entry.textContent = `${new Date().toLocaleTimeString()}: ${msg}`;
                    log.appendChild(entry);
                });
                log.scrollTop = log.scrollHeight;
            }
        }

        // Track offset so next poll only fetches new entries
        if (typeof task.logOffset === 'number') {
            this.logOffset = task.logOffset;
        }
    }

    updateSteps(steps) {
        for (const [stepName, stepStatus] of Object.entries(steps)) {
            const stepElement = document.querySelector(`[data-step="${stepName}"]`);
            if (stepElement) {
                stepElement.classList.remove('active', 'completed');
                if (stepStatus === 'running' || stepStatus === 'active') {
                    stepElement.classList.add('active');
                } else if (stepStatus === 'completed') {
                    stepElement.classList.add('completed');
                }
            }
        }
    }

    showProgress() {
        const progressPanel = document.getElementById('buildProgress');
        const buildLogs = document.getElementById('buildLogs');

        if (progressPanel) progressPanel.style.display = 'block';
        if (buildLogs) buildLogs.style.display = 'block';

        // Disable + relabel while building (covers both local and
        // download source modes — for download, the source ISO is
        // fetched server-side during acquireSource, so the user must
        // not click again).
        if (window.setGenerateButtonDisabled) {
            window.setGenerateButtonDisabled(true, 'Building ISO...');
        } else {
            const generateBtn = document.getElementById('generateBtn');
            if (generateBtn) {
                generateBtn.disabled = true;
                generateBtn.textContent = 'Building ISO...';
            }
        }

        // Reset steps
        document.querySelectorAll('.step').forEach(step => {
            step.classList.remove('active', 'completed');
        });

        // Clear previous build logs and reset offset
        const logContainer = document.getElementById('logContainer');
        if (logContainer) logContainer.innerHTML = '';
        this.logOffset = 0;

        // Reset progress bar
        const fill = document.getElementById('progressFill');
        if (fill) fill.style.width = '0%';

        // Clear logs
        const log = document.getElementById('logContainer');
        if (log) log.innerHTML = '';
    }

    hideProgress() {
        const progressPanel = document.getElementById('buildProgress');
        const buildLogs = document.getElementById('buildLogs');

        if (progressPanel) progressPanel.style.display = 'none';
        if (buildLogs) buildLogs.style.display = 'none';

        if (window.setGenerateButtonDisabled) {
            window.setGenerateButtonDisabled(false);
        } else {
            const generateBtn = document.getElementById('generateBtn');
            if (generateBtn) {
                generateBtn.disabled = false;
                generateBtn.textContent = 'Generate ISO Image';
            }
        }
    }

    updateProgress(percent, message) {
        const fill = document.getElementById('progressFill');
        const text = document.getElementById('progressText');
        const log = document.getElementById('logContainer');

        if (fill) {
            fill.style.width = `${percent}%`;
        }
        if (text) {
            text.textContent = `${percent}% - ${message}`;
        }
        if (log) {
            log.innerHTML += `<div>${new Date().toLocaleTimeString()}: ${message}</div>`;
            log.scrollTop = log.scrollHeight;
        }
    }

    updateProgressError(message) {
        this.updateProgress(0, `Error: ${message}`);
        this.hideProgress();
    }

    onComplete(task) {
        this.updateProgress(100, 'ISO generation completed!');
        window.showToast('ISO generated successfully!', 'success');

        const generateBtn = document.getElementById('generateBtn');
        if (generateBtn) {
            generateBtn.disabled = false;
            generateBtn.textContent = 'Generate ISO Image';
        }

        const downloadSection = document.getElementById('downloadSection');
        if (downloadSection) downloadSection.style.display = 'block';
    }

    async downloadISO(distro, url) {
        try {
            const response = await fetch('/api/iso/download', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ distro, url })
            });

            const data = await response.json();
            if (data.success) {
                window.showToast('ISO download started', 'info');
            }
        } catch (error) {
            window.showToast('Failed to download ISO', 'error');
        }
    }

    downloadCurrentISO() {
        if (!this.currentTask) {
            window.showToast('No ISO available for download', 'error');
            return;
        }
        window.location.href = `/api/iso/download/${this.currentTask}`;
    }
}

// Export
window.ISOGenerator = ISOGenerator;
