// Templates Manager Module
// Handles template loading, saving, import/export with left-right layout

class TemplatesManager {
    constructor() {
        this.templates = [];
        this.loaded = false;
        this.selectedTemplate = null;
        this.importSource = null;
        this.pendingKSContent = null;
        this.init();
    }

    init() {
        // Reserved for future non-DOM initialization hooks
    }

    async loadTemplates() {
        console.log('[TemplatesManager] loadTemplates called');
        try {
            const response = await fetch('/api/templates');
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }
            const data = await response.json();
            console.log('[TemplatesManager] API response:', data);
            console.log('[TemplatesManager] data.success:', data.success);
            console.log('[TemplatesManager] data.data:', data.data);
            console.log('[TemplatesManager] data.data length:', data.data ? data.data.length : 'undefined');

            if (data.data && data.data.length > 0) {
                console.log('[TemplatesManager] First template:', data.data[0]);
                console.log('[TemplatesManager] First template type:', data.data[0].type);
            }

            if (data.success) {
                this.templates = data.data || [];
                console.log('[TemplatesManager] templates loaded:', this.templates.length);
                console.log('[TemplatesManager] template types:', this.templates.map(t => t.type));
                this.renderTemplateList();
                this.loaded = true;

                if (this.templates.length > 0) {
                    if (!this.selectedTemplate) {
                        // Prefer the dedicated "default" template (loaded from
                        // templates/presets/default.cfg) so the Templates page
                        // matches the Configuration form auto-loaded via
                        // /api/config/default. Fall back to the first preset,
                        // then to any template, if no "default" exists.
                        const defaultTemplate =
                            this.templates.find(t => t.name === 'default') ||
                            this.templates.find(t => t.type === 'preset') ||
                            this.templates[0];
                        console.log('[TemplatesManager] defaultTemplate:', defaultTemplate);
                        if (defaultTemplate) {
                            this.selectTemplate(defaultTemplate);
                        }
                    }
                } else {
                    this.updateActionButtons(false);
                    this.clearPreview();
                }
            } else {
                console.error('[TemplatesManager] API returned success=false:', data.error);
                this.showError(data.error || 'Failed to load templates');
            }
        } catch (error) {
            console.error('[TemplatesManager] Failed to load templates:', error);
            this.showError(error.message || 'Failed to load templates');
        }
    }

    renderTemplateList() {
        console.log('[TemplatesManager] renderTemplateList called');
        const presetList = document.getElementById('presetTemplatesList');
        const userList = document.getElementById('userTemplatesList');

        console.log('[TemplatesManager] presetList element:', presetList);
        console.log('[TemplatesManager] userList element:', userList);

        // Clear lists
        if (presetList) presetList.innerHTML = '';
        if (userList) userList.innerHTML = '';

        const presets = this.templates.filter(t => t.type === 'preset');
        const users = this.templates.filter(t => t.type === 'user');
        console.log('[TemplatesManager] presets count:', presets.length);
        console.log('[TemplatesManager] users count:', users.length);

        // Render presets
        if (presets.length === 0) {
            if (presetList) {
                console.log('[TemplatesManager] No presets, showing empty message');
                presetList.innerHTML = '<div class="template-empty">No system templates</div>';
            }
        } else {
            console.log('[TemplatesManager] Rendering', presets.length, 'presets');
            presets.forEach(template => {
                const item = this.createTemplateItem(template);
                if (presetList) presetList.appendChild(item);
            });
        }

        // Render user templates
        if (users.length === 0) {
            if (userList) {
                console.log('[TemplatesManager] No users, showing empty message');
                userList.innerHTML = '<div class="template-empty">No custom templates</div>';
            }
        } else {
            console.log('[TemplatesManager] Rendering', users.length, 'user templates');
            users.forEach(template => {
                const item = this.createTemplateItem(template);
                if (userList) userList.appendChild(item);
            });
        }
    }

    createTemplateItem(template) {
        const item = document.createElement('div');
        item.className = `template-item ${this.selectedTemplate?.name === template.name ? 'selected' : ''}`;
        item.dataset.name = template.name;

        item.innerHTML = `
            <div class="template-item-content">
                <div class="template-item-name">${template.name}</div>
            </div>
        `;

        item.addEventListener('click', () => this.selectTemplate(template));

        return item;
    }

    selectTemplate(template) {
        // Update selection
        this.selectedTemplate = template;

        // Update UI selection
        document.querySelectorAll('.template-item').forEach(item => {
            item.classList.remove('selected');
        });
        document.querySelector(`.template-item[data-name="${template.name}"]`)?.classList.add('selected');

        // Update preview
        this.renderPreview(template);

        // Enable action buttons
        this.updateActionButtons(true);
    }

    renderPreview(template) {
        const previewContainer = document.getElementById('templatePreview');
        const titleEl = document.getElementById('previewTitle');

        if (!previewContainer) return;

        if (titleEl) {
            titleEl.textContent = 'Template Preview';
        }

        // Display raw Kickstart content
        const rawContent = template.rawContent || template.config?.rawContent || 'No raw content available';

        previewContainer.innerHTML = `
            <pre class="preview-yaml">${this.escapeHtml(rawContent)}</pre>
        `;
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    getTemplatePreviewContent(template) {
        if (!template.config) {
            return 'No configuration data available';
        }

        const config = template.config;
        const lines = [];

        // Install source
        if (config.method) {
            if (config.method.URL) lines.push(`Source: URL (${config.method.URL})`);
            else if (config.method.CDROM) lines.push(`Source: CDROM`);
            else if (config.method.NFS) lines.push(`Source: NFS`);
        }

        // Install mode
        if (config.installMode) {
            lines.push(`Install Mode: ${config.installMode}`);
        }

        // Basic info
        if (config.locale) {
            if (config.locale.timezone) lines.push(`Timezone: ${config.locale.timezone}`);
            if (config.locale.language) lines.push(`Language: ${config.locale.language}`);
            if (config.locale.keyboard) lines.push(`Keyboard: ${config.locale.keyboard}`);
        }

        // Storage
        if (config.storage) {
            if (config.storage.autopart) {
                lines.push(`Storage: Autopart (${config.storage.autopartType || 'lvm'})`);
            }
            if (config.storage.partitions && config.storage.partitions.length > 0) {
                const parts = config.storage.partitions;
                const mounts = parts.filter(p => p.mountpoint || p.Mountpoint).map(p => p.mountpoint || p.Mountpoint);
                lines.push(`Partitions: ${parts.length} (${mounts.join(', ')})`);
            }
            if (config.storage.volGroups && config.storage.volGroups.length > 0) {
                const vgs = config.storage.volGroups.map(v => v.name || v.Name || 'vg');
                lines.push(`Volume Groups: ${config.storage.volGroups.length} (${vgs.join(', ')})`);
            }
            if (config.storage.logVols && config.storage.logVols.length > 0) {
                lines.push(`Logical Volumes: ${config.storage.logVols.length}`);
            }
            if (config.storage.zerombr) lines.push(`Zerombr: enabled`);
            if (config.storage.clearAll) lines.push(`Clear All: enabled`);
        }

        // Packages
        if (config.packages) {
            if (config.packages.groups && config.packages.groups.length > 0) {
                const groupNames = config.packages.groups.map(g => g.name || g.Name).filter(n => n);
                lines.push(`Package Groups: ${config.packages.groups.length}`);
                lines.push(`  ${groupNames.slice(0, 3).join(', ')}${groupNames.length > 3 ? '...' : ''}`);
            }
            if (config.packages.packages && config.packages.packages.length > 0) {
                lines.push(`Packages: ${config.packages.packages.length}`);
            }
        }

        // Network
        if (config.networks && config.networks.length > 0) {
            lines.push(`Network: ${config.networks.length} device(s)`);
        }

        // Firewall
        if (config.firewall) {
            lines.push(`Firewall: ${config.firewall.disabled ? 'Disabled' : 'Enabled'}`);
        }

        // SELinux
        if (config.selinux) {
            lines.push(`SELinux: ${config.selinux.disabled ? 'Disabled' : 'Enabled'}`);
        }

        // Custom sections
        if (config.customSections && Object.keys(config.customSections).length > 0) {
            const sectionNames = Object.keys(config.customSections).map(name => name === 'anaconda' ? '%anaconda' : name);
            lines.push(`Custom Sections: ${sectionNames.length}`);
            lines.push(`  ${sectionNames.slice(0, 3).join(', ')}${sectionNames.length > 3 ? '...' : ''}`);
        }

        // Scripts
        if (config.preScripts && config.preScripts.length > 0) {
            lines.push(`Pre-scripts: ${config.preScripts.length}`);
        }
        if (config.postScripts && config.postScripts.length > 0) {
            lines.push(`Post-scripts: ${config.postScripts.length}`);
        }
        if (config.postNoChrootScripts && config.postNoChrootScripts.length > 0) {
            lines.push(`Post-nochroot scripts: ${config.postNoChrootScripts.length}`);
        }

        if (lines.length === 0) {
            return 'Empty configuration';
        }

        return lines.join('\n');
    }

    updateActionButtons(enabled) {
        const buttons = ['applyTemplateBtn', 'exportTemplateBtn', 'copyTemplateBtn', 'deleteTemplateBtn'];
        buttons.forEach(id => {
            const btn = document.getElementById(id);
            if (btn) {
                btn.disabled = !enabled;
            }
        });

        const editBtn = document.getElementById('editTemplateBtn');
        const isUserTemplate = this.selectedTemplate?.type === 'user';
        if (editBtn) {
            // Show Edit button only for user templates
            editBtn.style.display = isUserTemplate ? '' : 'none';
            editBtn.disabled = !enabled || !isUserTemplate;
        }

        // Show/hide delete button based on template type
        const deleteBtn = document.getElementById('deleteTemplateBtn');
        if (deleteBtn) {
            deleteBtn.style.display = (this.selectedTemplate?.type === 'user') ? '' : 'none';
        }
    }

    // ============ Actions ============

    async applySelectedTemplate() {
        if (!this.selectedTemplate) return;

        try {
            const response = await fetch(`/api/templates/${this.selectedTemplate.name}`);
            const data = await response.json();

            if (data.success && data.data) {
                // Use config from template (data.data.config) or the whole data object
                const config = data.data.config || data.data;

                console.log('=== Apply Template Debug ===');
                console.log('API Response:', JSON.stringify(data, null, 2));
                console.log('Config extracted:', JSON.stringify(config, null, 2));

                window.AppState.config = config;
                window.updateFormFromConfig(config);
                window.showToast(`Template "${this.selectedTemplate.name}" applied`, 'success');

                // Switch to the Configuration page — uses the same AppNavigation
                // object that other pages use. The optional chaining guards against
                // any initialization-order edge cases.
                if (window.AppNavigation && typeof window.AppNavigation.switchPage === 'function') {
                    window.AppNavigation.switchPage('config');
                } else if (typeof AppNavigation !== 'undefined') {
                    // Fallback: direct module reference (same as UbuntuCraft)
                    AppNavigation.switchPage('config');
                } else {
                    console.error('[TemplatesManager] AppNavigation not available, page switch skipped');
                }
            } else {
                window.showToast(data.error || 'Failed to load template', 'error');
            }
        } catch (error) {
            console.error('Failed to load template:', error);
            window.showToast('Failed to load template', 'error');
        }
    }

    exportSelectedTemplate() {
        if (!this.selectedTemplate) {
            window.showToast('Please select a template first', 'warning');
            return;
        }

        const content = this.selectedTemplate.rawContent;
        if (!content) {
            window.showToast('No content to export', 'warning');
            return;
        }

        const blob = new Blob([content], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `${this.selectedTemplate.name}.cfg`;
        a.click();
        URL.revokeObjectURL(url);
        window.showToast('Template exported', 'success');
    }

    copySelectedTemplate() {
        if (!this.selectedTemplate) {
            window.showToast('Please select a template first', 'warning');
            return;
        }

        const content = this.selectedTemplate.rawContent || '';
        if (!content) {
            window.showToast('No content to copy', 'warning');
            return;
        }

        this.copyToClipboard(content);
        window.showToast('Content copied to clipboard', 'success');
    }

    // ============ Edit Template ============

    isEditing = false;
    originalContent = '';

    editSelectedTemplate() {
        if (!this.selectedTemplate || this.selectedTemplate.type !== 'user') {
            window.showToast('Only custom templates can be edited', 'warning');
            return;
        }

        const previewContainer = document.getElementById('templatePreview');
        const titleEl = document.getElementById('previewTitle');
        const previewActionsNormal = document.getElementById('previewActionsNormal');
        const previewActionsEdit = document.getElementById('previewActionsEdit');

        // Get raw content
        this.originalContent = this.selectedTemplate.rawContent || '';

        // Switch to edit mode
        this.isEditing = true;

        // Update title
        if (titleEl) {
            titleEl.textContent = `Editing: ${this.selectedTemplate.name}`;
        }

        // Show edit textarea
        if (previewContainer) {
            previewContainer.innerHTML = `
                <textarea id="editTemplateContent" class="code-editor edit-textarea">${this.escapeHtml(this.originalContent)}</textarea>
            `;
        }

        // Toggle action buttons
        if (previewActionsNormal) previewActionsNormal.style.display = 'none';
        if (previewActionsEdit) previewActionsEdit.style.display = 'flex';
    }

    cancelEdit() {
        const previewContainer = document.getElementById('templatePreview');
        const titleEl = document.getElementById('previewTitle');
        const previewActionsNormal = document.getElementById('previewActionsNormal');
        const previewActionsEdit = document.getElementById('previewActionsEdit');

        // Exit edit mode
        this.isEditing = false;

        // Restore title
        if (titleEl) {
            titleEl.textContent = 'Template Preview';
        }

        // Restore preview
        this.renderPreview(this.selectedTemplate);

        // Toggle action buttons
        if (previewActionsNormal) previewActionsNormal.style.display = 'flex';
        if (previewActionsEdit) previewActionsEdit.style.display = 'none';
    }

    async saveEdit() {
        if (!this.selectedTemplate || !this.isEditing) return;

        const editContent = document.getElementById('editTemplateContent')?.value;
        if (!editContent) {
            window.showToast('Template content is empty', 'warning');
            return;
        }

        try {
            // Parse the edited content
            const parseResponse = await fetch('/api/config/parse-ks', {
                method: 'POST',
                headers: { 'Content-Type': 'text/plain' },
                body: editContent
            });

            const parseData = await parseResponse.json();
            if (!parseData.success) {
                window.showToast(parseData.error || 'Failed to parse Kickstart content', 'error');
                return;
            }

            // Save the template with new content
            const saveResponse = await fetch(`/api/templates/${this.selectedTemplate.name}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    config: parseData.data,
                    rawContent: editContent
                })
            });

            const saveData = await saveResponse.json();
            if (saveData.success) {
                window.showToast(`Template "${this.selectedTemplate.name}" saved`, 'success');
                this.isEditing = false;

                // Update local cache
                this.selectedTemplate.rawContent = editContent;
                this.selectedTemplate.config = parseData.data;

                // Exit edit mode
                this.cancelEdit();
                await this.loadTemplates();

                // Reselect the template
                const updatedTemplate = this.templates.find(t => t.name === this.selectedTemplate.name);
                if (updatedTemplate) {
                    this.selectTemplate(updatedTemplate);
                }
            } else {
                window.showToast(saveData.error || 'Failed to save template', 'error');
            }
        } catch (error) {
            console.error('Failed to save template:', error);
            window.showToast('Failed to save template', 'error');
        }
    }

    deleteSelectedTemplate() {
        if (!this.selectedTemplate || this.selectedTemplate.type !== 'user') {
            return;
        }

        this.showDeleteConfirm();
    }

    showDeleteConfirm() {
        const modal = document.getElementById('deleteConfirmModal');
        const message = document.getElementById('deleteConfirmMessage');

        if (message) {
            message.textContent = `Are you sure you want to delete the template "${this.selectedTemplate.name}"?`;
        }

        if (modal) {
            modal.classList.add('show');
        }
    }

    closeDeleteConfirm() {
        const modal = document.getElementById('deleteConfirmModal');
        if (modal) {
            modal.classList.remove('show');
        }
    }

    async confirmDelete() {
        this.closeDeleteConfirm();

        try {
            const response = await fetch(`/api/templates/${this.selectedTemplate.name}`, {
                method: 'DELETE'
            });

            const data = await response.json();
            if (data.success) {
                window.showToast(`Template "${this.selectedTemplate.name}" deleted`, 'success');
                this.selectedTemplate = null;
                this.updateActionButtons(false);
                this.clearPreview();
                await this.loadTemplates();
            } else {
                window.showToast(data.error || 'Failed to delete template', 'error');
            }
        } catch (error) {
            console.error('Failed to delete template:', error);
            window.showToast('Failed to delete template', 'error');
        }
    }

    clearPreview() {
        const previewContainer = document.getElementById('templatePreview');
        const titleEl = document.getElementById('previewTitle');

        if (titleEl) titleEl.textContent = 'Template Preview';
        if (previewContainer) {
            previewContainer.innerHTML = `
                <div class="preview-placeholder">
                    <span class="placeholder-icon">&#128194;</span>
                    <p>Select a template to preview</p>
                </div>
            `;
        }
    }

    // ============ Save Modal ============

    showSaveModal() {
        const modal = document.getElementById('saveTemplateModal');
        const nameInput = document.getElementById('newTemplateName');

        if (nameInput) nameInput.value = '';
        if (modal) modal.classList.add('show');
    }

    closeSaveModal() {
        const modal = document.getElementById('saveTemplateModal');
        if (modal) modal.classList.remove('show');
    }

    async confirmSaveTemplate() {
        const nameInput = document.getElementById('newTemplateName');
        const name = nameInput?.value?.trim();

        if (!name) {
            window.showToast('Please enter a template name', 'warning');
            return;
        }

        const config = window.getConfigFromForm();

        try {
            const response = await fetch('/api/templates', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name, config })
            });

            const data = await response.json();
            if (data.success) {
                window.showToast(`Template "${name}" saved`, 'success');
                this.closeSaveModal();
                await this.loadTemplates();
            } else {
                window.showToast(data.error || 'Failed to save template', 'error');
            }
        } catch (error) {
            console.error('Failed to save template:', error);
            window.showToast('Failed to save template', 'error');
        }
    }

    // ============ Import Modal ============

    // ============ Import Modal ============

    showImportModal() {
        const modal = document.getElementById('importTemplateModal');
        if (modal) modal.classList.add('show');

        this.resetImportModal();
        this.switchImportMode('paste');
        this.bindImportFileInput();
    }

    resetImportModal() {
        const nameInput = document.getElementById('importTemplateName');
        const yamlInput = document.getElementById('importTemplateYaml');
        const fileInput = document.getElementById('importFileInput');
        const filePreview = document.getElementById('importFilePreview');

        if (nameInput) nameInput.value = '';
        if (yamlInput) yamlInput.value = '';
        if (fileInput) fileInput.value = '';
        if (filePreview) filePreview.innerHTML = '';
    }

    switchImportMode(mode) {
        const pasteBtn = document.getElementById('importModePaste');
        const uploadBtn = document.getElementById('importModeUpload');
        const pasteMode = document.getElementById('importPasteMode');
        const uploadMode = document.getElementById('importUploadMode');

        if (pasteBtn) pasteBtn.classList.toggle('active', mode === 'paste');
        if (uploadBtn) uploadBtn.classList.toggle('active', mode === 'upload');
        if (pasteMode) pasteMode.style.display = mode === 'paste' ? 'block' : 'none';
        if (uploadMode) uploadMode.style.display = mode === 'upload' ? 'block' : 'none';
    }

    bindImportFileInput() {
        const fileInput = document.getElementById('importFileInput');
        if (!fileInput) return;

        fileInput.onchange = (e) => {
            const file = e.target.files?.[0];
            if (!file) return;
            this.handleImportFile(file);
        };
    }

    handleImportFile(file) {
        if (!file.name.toLowerCase().endsWith('.cfg')) {
            window.showToast('Please select a .cfg file', 'error');
            return;
        }

        const reader = new FileReader();
        reader.onload = (e) => {
            const content = e.target?.result || '';
            const yamlInput = document.getElementById('importTemplateYaml');
            const preview = document.getElementById('importFilePreview');

            if (yamlInput) {
                yamlInput.value = String(content);
            }
            if (preview) {
                preview.innerHTML = `<span class="import-file-info">Selected: ${this.escapeHtml(file.name)} (${this.formatFileSize(file.size)})</span>`;
            }
        };
        reader.onerror = () => {
            window.showToast('Failed to read file', 'error');
        };
        reader.readAsText(file);
    }

    formatFileSize(bytes) {
        if (bytes < 1024) return `${bytes} B`;
        if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
        return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    }

    async confirmImportTemplate() {
        const nameInput = document.getElementById('importTemplateName');
        const yamlInput = document.getElementById('importTemplateYaml');
        const fileInput = document.getElementById('importFileInput');
        const pasteBtn = document.getElementById('importModePaste');

        const isPasteMode = pasteBtn?.classList.contains('active');
        const content = yamlInput?.value?.trim();

        if (!content) {
            window.showToast('Kickstart content is required', 'error');
            return;
        }

        let name;
        if (isPasteMode) {
            name = nameInput?.value?.trim();
            if (!name) {
                window.showToast('Template name is required', 'error');
                return;
            }
        } else {
            const file = fileInput?.files?.[0];
            if (!file) {
                window.showToast('Please select a file to upload', 'error');
                return;
            }
            name = file.name.replace(/\.cfg$/i, '');
        }

        try {
            const response = await fetch('/api/config/parse-ks', {
                method: 'POST',
                headers: { 'Content-Type': 'text/plain' },
                body: content
            });

            const data = await response.json();
            if (data.success) {
                await this.saveImportedTemplate(name, data.data);
            } else {
                window.showToast(data.error || 'Failed to parse Kickstart file', 'error');
            }
        } catch (error) {
            console.error('Failed to import template:', error);
            window.showToast('Failed to import template', 'error');
        }
    }

    async saveImportedTemplate(name, config) {
        try {
            const response = await fetch('/api/templates', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name, config })
            });

            const data = await response.json();
            if (data.success) {
                window.showToast(`Template "${name}" imported`, 'success');
                this.closeImportModal();
                await this.loadTemplates();
            } else {
                window.showToast(data.error || 'Failed to save template', 'error');
            }
        } catch (error) {
            console.error('Failed to save template:', error);
            window.showToast('Failed to save template', 'error');
        }
    }

    closeImportModal() {
        const modal = document.getElementById('importTemplateModal');
        if (modal) modal.classList.remove('show');
    }

    // ============ Helpers ============

    showError(message) {
        const presetList = document.getElementById('presetTemplatesList');
        const userList = document.getElementById('userTemplatesList');
        const empty = '<div class="template-empty">Error loading templates</div>';

        if (presetList) presetList.innerHTML = empty;
        if (userList) userList.innerHTML = empty;
    }

    copyToClipboard(text) {
        if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(text).then(() => {
                // Success
            }).catch(err => {
                console.error('Clipboard API failed, using fallback:', err);
                this.fallbackCopy(text);
            });
        } else {
            this.fallbackCopy(text);
        }
    }

    fallbackCopy(text) {
        const textarea = document.createElement('textarea');
        textarea.value = text;
        textarea.style.position = 'fixed';
        textarea.style.opacity = '0';
        document.body.appendChild(textarea);
        textarea.select();
        try {
            document.execCommand('copy');
        } catch (err) {
            console.error('Fallback copy failed:', err);
        }
        document.body.removeChild(textarea);
    }
}

// Export
window.TemplatesManager = TemplatesManager;
