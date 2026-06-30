// KickCraft Main Application
// Global state and initialization

const API_BASE = '/api';

// parseCustomSectionBlock accepts raw text from the Addon textarea
// and returns {name, body} where name is the section header text
// without the leading "%" (e.g. "addon com_redhat_kdump --enable")
// and body is the lines between the opener and the matching %end.
//
// Returns null if the text is empty or does not start with a %
// section opener. The caller decides how to fall back.
function parseCustomSectionBlock(text) {
    if (!text) return null;
    const lines = text.split('\n');
    let first = -1;
    for (let i = 0; i < lines.length; i++) {
        if (lines[i].trim() !== '') {
            first = i;
            break;
        }
    }
    if (first < 0) return null;
    const opener = lines[first].trim();
    if (!opener.startsWith('%')) return null;
    // Find matching %end
    let end = -1;
    for (let i = first + 1; i < lines.length; i++) {
        if (lines[i].trim() === '%end') { end = i; break; }
    }
    const bodyLines = (end > first) ? lines.slice(first + 1, end) : lines.slice(first + 1);
    return {
        name: opener.slice(1).trim(), // drop leading "%"
        body: bodyLines.join('\n'),
    };
}

// Application State
const AppState = {
    distro: '',
    config: null,
    templates: [],
    currentPage: 'config',
    currentTab: 'basic',
    isDirty: false,
    validationStatus: 'unknown',
    errors: [],
    warnings: [],
    rawLines: []  // Preserve original command order from imported KS files
};

// Page Navigation (matching UbuntuCraft pattern)
const AppNavigation = {
    switchPage(pageName) {
        console.log('[AppNavigation] switchPage called with:', pageName);

        // Update sidebar active state
        document.querySelectorAll('.sidebar-item').forEach(item => {
            item.classList.remove('active');
            if (item.dataset.page === pageName) {
                item.classList.add('active');
            }
        });

        // Update content pages
        document.querySelectorAll('.content-page').forEach(page => {
            page.classList.remove('active');
            if (page.dataset.page === pageName) {
                page.classList.add('active');
            }
        });

        AppState.currentPage = pageName;

        // Load page-specific content
        if (pageName === 'templates') {
            console.log('[AppNavigation] Switching to templates page');
            console.log('[AppNavigation] templatesManager:', window.templatesManager);
            if (window.templatesManager) {
                window.templatesManager.loadTemplates();
            } else {
                console.error('[AppNavigation] templatesManager not found!');
            }
        }
        if (pageName === 'system-info' && window.SystemInfoManager) {
            window.SystemInfoManager.init();
        }

        console.log('Switched to page:', pageName);
    }
};

// Confirmation Modal — modeled after UbuntuCraft's showConfirmModal.
// Renders an inline confirm dialog with the given title/message and
// invokes onConfirm() if the user clicks Confirm. Cancel simply closes.
function showConfirmModal(title, message, onConfirm) {
    const overlay = document.createElement('div');
    overlay.className = 'confirm-modal-overlay';
    overlay.innerHTML = `
        <div class="modal-dialog" role="dialog" aria-modal="true">
            <div class="modal-header">
                <h3>${title}</h3>
            </div>
            <div class="modal-body">
                <p>${message}</p>
            </div>
            <div class="modal-footer">
                <button type="button" class="btn btn-secondary modal-cancel">Cancel</button>
                <button type="button" class="btn btn-primary modal-confirm">Confirm</button>
            </div>
        </div>
    `;
    document.body.appendChild(overlay);

    const close = () => {
        if (overlay.parentNode) overlay.parentNode.removeChild(overlay);
    };

    overlay.querySelector('.modal-cancel').addEventListener('click', close);
    overlay.addEventListener('click', (e) => {
        if (e.target === overlay) close();
    });
    overlay.querySelector('.modal-confirm').addEventListener('click', async () => {
        close();
        try {
            await onConfirm();
        } catch (err) {
            console.error('showConfirmModal onConfirm error:', err);
        }
    });
}

// Toast Notifications
function showToast(message, type = 'info', duration = 3000) {
    const container = document.getElementById('toastContainer');
    if (!container) return;
    
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.innerHTML = `
        <span class="toast-icon">${getToastIcon(type)}</span>
        <span class="toast-message">${message}</span>
    `;
    container.appendChild(toast);

    setTimeout(() => {
        toast.classList.add('toast-show');
    }, 10);

    setTimeout(() => {
        toast.classList.remove('toast-show');
        setTimeout(() => toast.remove(), 300);
    }, duration);
}

function getToastIcon(type) {
    switch (type) {
        case 'success': return '&#10004;';
        case 'error': return '&#10006;';
        case 'warning': return '&#9888;';
        default: return '&#8505;';
    }
}

/**
 * Show status message in an inline status element (under page actions)
 * Mirrors UbuntuCraft's showStatus — used to display success/error feedback
 * directly below the action buttons (e.g. #configActionStatus).
 */
function showStatus(elementId, type, message) {
    const element = document.getElementById(elementId);
    if (!element) return;

    element.className = `action-status status ${type}`;
    element.textContent = message;
    element.style.display = 'block';

    // Auto-hide success and info messages after 10 seconds; keep error/warning
    // visible until the next action so the user can read what went wrong.
    if (type === 'success' || type === 'info') {
        setTimeout(() => {
            element.style.display = 'none';
        }, 10000);
    }
}

// Modal Management
function openModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
        modal.style.display = 'flex';
    }
}

function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
        modal.style.display = 'none';
    }
}

// Tab Management (for content tabs)
function initTabs() {
    const tabs = document.querySelectorAll('.tab');
    const tabContents = document.querySelectorAll('.tab-content');

    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const tabId = tab.dataset.tab;

            // Update active tab
            tabs.forEach(t => t.classList.remove('active'));
            tab.classList.add('active');

            // Update active content
            tabContents.forEach(content => {
                content.classList.remove('active');
                if (content.id === tabId) {
                    content.classList.add('active');
                }
            });

            AppState.currentTab = tabId;

            // Trigger tab-activated hook for modules
            if (typeof window.onTabActivated === 'function') {
                window.onTabActivated(tabId);
            }
        });
    });
}

// Health Check
async function checkHealth() {
    const statusDot = document.getElementById('statusDot');
    try {
        const response = await fetch(`${API_BASE}/health`);
        const data = await response.json();
        if (data.success) {
            if (statusDot) {
                statusDot.className = 'status-dot status-ok';
            }
            showToast('Server is healthy', 'success');
        }
    } catch (error) {
        if (statusDot) {
            statusDot.className = 'status-dot status-error';
        }
        showToast('Server is not reachable', 'error');
    }
}

// Load Distros
async function loadDistros() {
    try {
        const response = await fetch(`${API_BASE}/distros`);
        const data = await response.json();
        if (data.success) {
            populateDistroSelect(data.data);
        }
    } catch (error) {
        console.error('Failed to load distros:', error);
    }
}

function populateDistroSelect(distros) {
    const select = document.getElementById('distroSelect');
    if (!select) return;
    
    // Clear existing options except first
    while (select.options.length > 1) {
        select.remove(1);
    }

    distros.forEach(distro => {
        const option = document.createElement('option');
        option.value = distro.id;
        option.textContent = distro.name;
        select.appendChild(option);
    });
}

// Configuration Management
function getConfigFromForm() {
    // Preserve rawLines from imported config
    const rawLines = AppState.rawLines || [];

    console.log('[getConfigFromForm] AppState.config:', JSON.stringify(window.AppState?.config, null, 2));

    const config = {
        rawLines: rawLines,
        locale: {
            lang: document.getElementById('langSelect')?.value || 'en_US.UTF-8',
            keymap: document.getElementById('keyboardSelect')?.value || 'us',
            timezone: document.getElementById('timezoneSelect')?.value || 'Asia/Shanghai',
            utc: document.getElementById('timezoneUTC')?.checked ?? false,
            nontp: document.getElementById('timezoneNontp')?.checked ?? false,
            hostname: document.getElementById('systemHostname')?.value || ''
        },
        // Installation source (cdrom, nfs, url, harddrive, etc.)
        method: {
            type: document.getElementById('installSourceType')?.value || 'cdrom',
            server: document.getElementById('nfsServer')?.value || '',
            dir: document.getElementById('nfsDir')?.value || '',
            url: document.getElementById('urlMirror')?.value || '',
            partition: document.getElementById('hdPartition')?.value || ''
        },
        // Installation mode (graphical, text, cmdline)
        installMode: document.getElementById('installMode')?.value || 'graphical',
        rootPassword: {
            password: document.getElementById('rootPassword')?.value || '',
            isCrypted: document.getElementById('rootPasswordCrypted')?.checked || false,
            lock: document.getElementById('lockRoot')?.checked || false,
            allowSsh: document.getElementById('rootPasswordAllowSsh')?.checked || false,
            isSet: !!document.getElementById('rootPassword')?.value
        },
        // EULA agreement
        eula: document.getElementById('eulaAgreed')?.checked ? 'agreed' : '',
        bootloader: {
            location: document.getElementById('bootloaderLocation')?.value || 'mbr',
            append: document.getElementById('bootloaderAppend')?.value || '',
            bootDrive: document.getElementById('bootloaderBootDrive')?.value || ''
        },
        firewall: {
            enabled: document.getElementById('firewallEnabled')?.checked ?? true,
            services: Array.from(document.querySelectorAll('.firewall-service:checked'))
                .map(el => el.value),
            ports: parseCommaSeparated('firewallPorts')
        },
        selinux: {
            mode: document.getElementById('selinuxMode')?.value || 'enforcing'
        },
        services: {
            enabled: (document.getElementById('servicesEnabled')?.value || '').split(',').filter(s => s.trim()),
            disabled: (document.getElementById('servicesDisabled')?.value || '').split(',').filter(s => s.trim())
        },
        kdump: {},
        storage: {
            zerombr: document.getElementById('zerombr')?.checked || false,
            clearAll: document.getElementById('clearpartAll')?.checked || false,
            initLabel: document.getElementById('initLabel')?.checked || false,
            autopart: document.querySelector('input[name="partitionMode"]:checked')?.value === 'auto',
            autopartType: document.getElementById('autopartType')?.value || 'lvm',
            partitions: [],
            raids: [],
            volGroups: [],
            logVols: [],
            btrfs: []
        },
        graphics: {
            skipX: document.getElementById('skipX')?.checked || false,
            firstBoot: document.getElementById('firstBootSelect')?.value || ''
        },
        powerAction: document.getElementById('powerAction')?.value || '',
        packages: {
            packages: parseLineSeparated('packagesList'),
            groups: Array.from(document.querySelectorAll('.package-group:checked')).map(cb => {
                const parsed = parsePackageGroupValue(cb.value);
                return { name: parsed.name, optional: parsed.optional };
            })
        },
        additionalPackages: parseLineSeparated('additionalPackages'),
        repos: [],
        preScripts: [],
        postScripts: [],
        postNoChrootScripts: []
    };

    // Parse ignoredisk options
    const enableIgnoredisk = document.getElementById('enableIgnoredisk')?.checked;
    if (enableIgnoredisk) {
        const ignorediskMode = document.querySelector('input[name="ignorediskMode"]:checked')?.value;
        if (ignorediskMode === 'ignore') {
            config.storage.ignoreDiskDrives = parseCommaSeparated('ignorediskDrives');
        } else if (ignorediskMode === 'onlyuse') {
            config.storage.onlyUseDrives = parseCommaSeparated('onlyuseDrives');
        }
    }

    // Sync storage from StorageManager to AppState.config, then read it back.
    // This ensures StorageManager's DOM-based state is consolidated into
    // AppState.config before we build the output config.
    if (window.StorageManager) {
        window.StorageManager.syncStorageConfig?.();

        // Read from the shared AppState.config (the single source of truth,
        // matching UbuntuCraft's pattern). This also handles the case where
        // StorageManager.syncStorageConfig was a no-op (e.g. before app init)
        // by reading whatever is in AppState.config.storage.
        if (window.AppState?.config?.storage) {
            config.storage = window.AppState.config.storage;
        }
    }

    // ============ Parse Additional Users ============
    const usersList = document.getElementById('usersList');
    if (usersList) {
        config.users = [];
        usersList.querySelectorAll('.user-item').forEach(item => {
            const name = item.querySelector('.user-name')?.value?.trim();
            if (name) {
                const user = {
                    name: name,
                    password: item.querySelector('.user-password')?.value || '',
                    shell: item.querySelector('.user-shell')?.value || '/bin/bash',
                    gecos: item.querySelector('.user-gecos')?.value || '',
                    groups: (item.querySelector('.user-groups')?.value || '').split(',').map(s => s.trim()).filter(s => s),
                    lock: item.querySelector('.user-lock')?.checked || false,
                    isPlaintext: item.querySelector('.user-plaintext')?.checked || false,
                    isCrypted: item.querySelector('.user-iscrypted')?.checked || false
                };
                config.users.push(user);
            }
        });
    }

    // Sync networks from NetworkManager to AppState.config, then read it back.
    // NetworkManager.updateDevice() writes each card's values into
    // window.AppState.config.networks[index] on every input change, so
    // reading from AppState.config gives us the latest state without
    // needing to re-query DOM selectors.
    if (window.networkManager && window.AppState?.config) {
        config.networks = window.AppState.config.networks || [];
    }

    // ============ Scripts Tab ============
    const preScriptContent = document.getElementById('preScript')?.value?.trim() || '';
    if (preScriptContent) {
        config.preScripts = [{
            type: 'pre',
            content: preScriptContent,
            interpreter: '/bin/bash',
            noChroot: true,
            errorOnFail: false
        }];
    }

    const postScriptContent = document.getElementById('postScript')?.value?.trim() || '';
    if (postScriptContent) {
        config.postScripts = [{
            type: 'post',
            content: postScriptContent,
            interpreter: '/bin/bash',
            noChroot: false,
            errorOnFail: false,
            log: '/mnt/sysimage/var/log/kickcraft-post.log'
        }];
    }

    const postNoChrootScriptContent = document.getElementById('postNoChrootScript')?.value?.trim() || '';
    if (postNoChrootScriptContent) {
        // Backend struct field is PostScriptsNoChroot (json:
        // "postScriptsNoChroot"). Earlier code used postNoChrootScripts
        // which was silently dropped on the way to ToString().
        config.postScriptsNoChroot = [{
            type: 'post',
            content: postNoChrootScriptContent,
            interpreter: '/bin/bash',
            noChroot: true,
            errorOnFail: false,
            log: '/mnt/sysimage/var/log/kickcraft-stage1.log'
        }];
    }

    // ============ Kdump (Addon) Section ============
    const kdumpEnabled = document.getElementById('kdumpEnabled')?.checked || false;
    if (kdumpEnabled) {
        const kdumpReserve = document.getElementById('kdumpReserve')?.value || 'auto';
        config.kdump = {
            enabled: true,
            reserveMb: kdumpReserve
        };
    } else {
        config.kdump = {
            enabled: false
        };
    }

    // ============ Repos Tab ============
    // The repo UI renders each repository as a .repo-card containing
    // <input class="repo-name-input"> and <input class="repo-url-input">.
    // Earlier code looked for .repo-item / .repo-item-name / data-url
    // which do not exist, so the form values were never sent.
    const repoList = document.getElementById('repoList');
    if (repoList) {
        config.repos = [];
        repoList.querySelectorAll('.repo-card').forEach(card => {
            const name = card.querySelector('.repo-name-input')?.value?.trim();
            const url = card.querySelector('.repo-url-input')?.value?.trim() || '';
            if (name) {
                config.repos.push({
                    name: name,
                    baseurl: url,
                    cost: 0
                });
            }
        });
    }

    return config;
}

// Helper function to parse comma-separated values
function parseCommaSeparated(elementId) {
    const element = document.getElementById(elementId);
    if (!element || !element.value) return [];
    return element.value.split(',').map(s => s.trim()).filter(s => s);
}

// Helper function to parse line-separated values (one per line)
function parseLineSeparated(elementId) {
    const element = document.getElementById(elementId);
    if (!element || !element.value) return [];
    return element.value.split('\n').map(s => s.trim()).filter(s => s);
}

// Helper function to parse package group value (e.g., "@^minimal-environment" or "@development")
function parsePackageGroupValue(value) {
    const isOptional = !value.startsWith('@^');
    const name = value.replace(/^@\^?/, '');
    return { name, optional: isOptional };
}

function updateFormFromConfig(config) {
    if (!config) {
        console.error('updateFormFromConfig: config is null or undefined');
        return;
    }

    console.log('=== updateFormFromConfig ===');
    console.log('Config keys:', Object.keys(config));
    console.log('Locale:', config.locale);
    console.log('Storage:', config.storage);
    console.log('RootPassword:', config.rootPassword);
    console.log('Users:', config.users);
    console.log('Networks:', config.networks);

    // Save rawLines for preserving command order
    if (config.rawLines && Array.isArray(config.rawLines)) {
        AppState.rawLines = config.rawLines;
        console.log('[updateFormFromConfig] Saved rawLines count:', config.rawLines.length);
    } else {
        AppState.rawLines = [];
    }

    // ============ Basic Configuration Tab ============

    // Locale settings
    if (config.locale) {
        setFormValue('langSelect', config.locale.lang || 'en_US.UTF-8');
        setFormValue('keyboardSelect', config.locale.keymap || 'us');
        setFormValue('timezoneSelect', config.locale.timezone || 'Asia/Shanghai');
        setFormChecked('timezoneUTC', config.locale.utc || false);
        setFormChecked('timezoneNontp', config.locale.nontp || false);
        setFormValue('systemHostname', config.locale.hostname || '');
    }

    // Installation Source
    if (config.method) {
        setFormValue('installSourceType', config.method.type || 'cdrom');
        setFormValue('nfsServer', config.method.server || '');
        setFormValue('nfsDir', config.method.dir || '');
        setFormValue('urlMirror', config.method.url || '');
        setFormValue('hdPartition', config.method.partition || '');
        toggleInstallSourceOptions();
    }

    // Root Password
    if (config.rootPassword) {
        setFormValue('rootPassword', config.rootPassword.password || '');
        setFormChecked('rootPasswordCrypted', config.rootPassword.isCrypted || false);
        setFormChecked('lockRoot', config.rootPassword.lock || false);
        setFormChecked('rootPasswordAllowSsh', config.rootPassword.allowSsh || false);
    }

    // Additional Users
    if (config.users && Array.isArray(config.users)) {
        loadUsersFromConfig(config.users);
    }

    // Install Mode
    if (config.installMode) {
        setFormValue('installMode', config.installMode);
    }

    // First Boot
    if (config.graphics && config.graphics.firstBoot) {
        setFormValue('firstBootSelect', config.graphics.firstBoot);
    }

    // Skip X
    if (config.graphics) {
        setFormChecked('skipX', config.graphics.skipX || false);
    }

    // ============ Storage Tab ============
    // Check if storage config exists, if not create a default one
    if (!config.storage) {
        config.storage = {};
    }

    // Disk options
    setFormChecked('zerombr', config.storage.zerombr || false);
    setFormChecked('clearpartAll', config.storage.clearAll !== false);
    setFormChecked('initLabel', config.storage.initLabel || false);

    // Partition mode
    if (config.storage.autopart || config.storage.autopart === undefined) {
        const autoRadio = document.querySelector('input[name="partitionMode"][value="auto"]');
        if (autoRadio) autoRadio.checked = true;
    } else {
        const manualRadio = document.querySelector('input[name="partitionMode"][value="manual"]');
        if (manualRadio) manualRadio.checked = true;
    }
    togglePartitionMode();

    // Autopart type
    setFormValue('autopartType', config.storage.autopartType || 'lvm');

    // Load storage config into StorageManager
    if (window.StorageManager) {
        window.StorageManager.loadConfig(config.storage);
    }

    // ============ Network Tab ============
    if (config.networks && Array.isArray(config.networks)) {
        loadNetworksFromConfig(config.networks);
    }

    // ============ Packages Tab ============
    if (config.packages) {
        // Load package groups - reset all first
        const groupCheckboxes = document.querySelectorAll('.package-group');
        groupCheckboxes.forEach(cb => cb.checked = false);

        // Then set checked based on config
        if (config.packages.groups && Array.isArray(config.packages.groups)) {
            config.packages.groups.forEach(g => {
                const groupName = g.name.replace(/^@\^?/, '');
                groupCheckboxes.forEach(cb => {
                    const checkboxName = cb.value.replace(/^@\^?/, '');
                    if (checkboxName === groupName) {
                        cb.checked = true;
                    }
                });
            });
        }

        // Load package list (one per line)
        if (config.packages.packages && Array.isArray(config.packages.packages)) {
            setFormValue('packagesList', config.packages.packages.join('\n'));
        }
        // Load additional packages list
        if (config.additionalPackages && Array.isArray(config.additionalPackages)) {
            setFormValue('additionalPackages', config.additionalPackages.join('\n'));
        }
    }

    // ============ Repos Section ============
    loadReposFromConfig(config.repos || []);

    // ============ Scripts Tab ============
    if (config.preScripts && config.preScripts.length > 0) {
        setFormValue('preScript', config.preScripts[0].content || '');
    }
    if (config.postScripts && config.postScripts.length > 0) {
        setFormValue('postScript', config.postScripts[0].content || '');
    }
    if (config.postScriptsNoChroot && config.postScriptsNoChroot.length > 0) {
        setFormValue('postNoChrootScript', config.postScriptsNoChroot[0].content || '');
    }
    if (config.customSections && typeof config.customSections === 'object') {
        setFormValue('anacondaSection', config.customSections.anaconda || '');
    }

    // ============ Kdump Tab ============
    // Check if kdump is configured either directly (config.kdump) or
    // via customSections (legacy template compatibility)
    let kdumpConfig = config.kdump || { enabled: false };

    // Parse kdump from customSections for backward compatibility with old templates
    if (config.customSections && typeof config.customSections === 'object') {
        for (const [key, value] of Object.entries(config.customSections)) {
            if (key.includes('com_redhat_kdump')) {
                kdumpConfig = { enabled: key.includes('--enable'), reserveMb: 'auto' };
                // Try to extract reserve-mb value
                const reserveMatch = key.match(/--reserve-mb\s*=\s*['"]([^'"]+)['"]/);
                if (reserveMatch) {
                    kdumpConfig.reserveMb = reserveMatch[1];
                }
                break;
            }
        }
    }

    if (kdumpConfig.enabled) {
        setFormChecked('kdumpEnabled', true);
        const kdumpSettings = document.getElementById('kdumpSettings');
        if (kdumpSettings) {
            kdumpSettings.style.display = 'block';
        }
        setFormValue('kdumpReserve', kdumpConfig.reserveMb || 'auto');
    } else {
        setFormChecked('kdumpEnabled', false);
        const kdumpSettings = document.getElementById('kdumpSettings');
        if (kdumpSettings) {
            kdumpSettings.style.display = 'none';
        }
    }

    // ============ Advanced Tab ============

    // Bootloader
    if (config.bootloader) {
        setFormValue('bootloaderLocation', config.bootloader.location || 'mbr');
        setFormValue('bootloaderBootDrive', config.bootloader.bootDrive || '');
        setFormValue('bootloaderAppend', config.bootloader.append || '');
    }

    // Firewall
    if (config.firewall) {
        setFormChecked('firewallEnabled', config.firewall.enabled !== false);

        // Firewall services - always reset all checkboxes first
        const serviceCheckboxes = document.querySelectorAll('.firewall-service');
        serviceCheckboxes.forEach(cb => {
            if (config.firewall.services && Array.isArray(config.firewall.services)) {
                cb.checked = config.firewall.services.includes(cb.value);
            } else {
                cb.checked = false;
            }
        });

        setFormValue('firewallPorts', config.firewall.ports ? config.firewall.ports.join(', ') : '');
    }

    // SELinux
    if (config.selinux) {
        setFormValue('selinuxMode', config.selinux.mode || 'enforcing');
    }

    // Power Action
    setFormValue('powerAction', config.powerAction || '');

    // EULA
    if (config.eula === 'agreed') {
        setFormChecked('eulaAgreed', true);
    }

        // Services
        document.getElementById('servicesEnabled').value = '';
        document.getElementById('servicesDisabled').value = '';
        if (config.services) {
            if (config.services.enabled && Array.isArray(config.services.enabled)) {
                setFormValue('servicesEnabled', config.services.enabled.join(', '));
            }
            if (config.services.disabled && Array.isArray(config.services.disabled)) {
                setFormValue('servicesDisabled', config.services.disabled.join(', '));
            }
        }

    // Update AppState
    window.AppState.config = config;
    window.AppState.isDirty = false;
}

// Helper functions
function setFormValue(id, value) {
    const el = document.getElementById(id);
    if (el) el.value = value;
}

function setFormChecked(id, checked) {
    const el = document.getElementById(id);
    if (el) el.checked = checked;
}

// ============ Load Users ============
function loadUsersFromConfig(users) {
    const usersList = document.getElementById('usersList');
    const template = document.getElementById('userItemTemplate');
    if (!usersList) return;

    // Clear existing users
    usersList.innerHTML = '';

    if (!users || !Array.isArray(users) || users.length === 0) return;

    users.forEach(user => {
        let userItem;
        if (template) {
            const clone = template.content.cloneNode(true);
            userItem = clone.querySelector('.user-item');
        } else {
            userItem = document.createElement('div');
            userItem.className = 'user-item';
            userItem.innerHTML = createUserItemHTML();
        }

        // Fill in user data
        userItem.querySelector('.user-name').value = user.name || '';
        userItem.querySelector('.user-password').value = user.password || '';
        userItem.querySelector('.user-shell').value = user.shell || '/bin/bash';

        // New fields
        const gecosInput = userItem.querySelector('.user-gecos');
        if (gecosInput) gecosInput.value = user.gecos || '';

        const groupsInput = userItem.querySelector('.user-groups');
        if (groupsInput) {
            // Handle both array and string formats
            if (Array.isArray(user.groups)) {
                groupsInput.value = user.groups.join(', ');
            } else {
                groupsInput.value = user.groups || '';
            }
        }

        const lockCheckbox = userItem.querySelector('.user-lock');
        if (lockCheckbox) lockCheckbox.checked = user.lock || false;

        const plaintextCheckbox = userItem.querySelector('.user-plaintext');
        if (plaintextCheckbox) plaintextCheckbox.checked = user.isPlaintext || false;

        const isCryptedCheckbox = userItem.querySelector('.user-iscrypted');
        if (isCryptedCheckbox) isCryptedCheckbox.checked = user.isCrypted || false;

        // Bind delete button
        userItem.querySelector('.user-delete').addEventListener('click', () => {
            userItem.remove();
        });

        usersList.appendChild(userItem);
    });
}

function createUserItemHTML() {
    return `
        <div class="form-row">
            <div class="form-group">
                <label>Username</label>
                <input type="text" class="form-input user-name" placeholder="username">
            </div>
            <div class="form-group">
                <label>Password</label>
                <input type="password" class="form-input user-password" placeholder="password">
            </div>
            <div class="form-group">
                <label>Shell</label>
                <select class="form-select user-shell">
                    <option value="/bin/bash">/bin/bash</option>
                    <option value="/bin/sh">/bin/sh</option>
                    <option value="/sbin/nologin">/sbin/nologin</option>
                </select>
            </div>
        </div>
        <div class="form-row">
            <div class="form-group" style="flex: 1;">
                <label class="no-required">GECOS <span class="hint-icon" data-tooltip="Full name, office, etc.">?</span></label>
                <input type="text" class="form-input user-gecos" placeholder="Full Name,Office,etc">
            </div>
            <div class="form-group" style="flex: 1;">
                <label class="no-required">Groups <span class="hint-icon" data-tooltip="Comma-separated group names">?</span></label>
                <input type="text" class="form-input user-groups" placeholder="wheel,docker">
            </div>
        </div>
        <div class="form-row">
            <div class="switch-container">
                <label class="switch">
                    <input type="checkbox" class="user-lock">
                    <span class="slider"></span>
                </label>
                <span class="switch-label">Lock account <span class="hint-icon" data-tooltip="Lock the user account">?</span></span>
            </div>
            <div class="switch-container">
                <label class="switch">
                    <input type="checkbox" class="user-plaintext">
                    <span class="slider"></span>
                </label>
                <span class="switch-label">Plaintext password <span class="hint-icon" data-tooltip="Store password as plaintext, not encrypted">?</span></span>
            </div>
            <div class="switch-container">
                <label class="switch">
                    <input type="checkbox" class="user-iscrypted">
                    <span class="slider"></span>
                </label>
                <span class="switch-label">Is encrypted <span class="hint-icon" data-tooltip="Password is already encrypted (SHA512)">?</span></span>
            </div>
            <button type="button" class="btn btn-sm btn-danger user-delete" style="margin-left: auto;">Delete</button>
        </div>
    `;
}

// ============ Load Networks ============
function loadNetworksFromConfig(networks) {
    // Use networkManager instance (lowercase) which is initialized in init().
    // window.NetworkManager (uppercase) is the class itself, not an instance.
    if (window.networkManager && typeof window.networkManager.loadConfig === 'function') {
        window.networkManager.loadConfig(networks);
        return;
    }
    
    // Fallback: clear existing networks and create basic network cards
    const networkDevices = document.getElementById('networkDevices');
    if (!networkDevices) return;
    
    networkDevices.innerHTML = '';
    
    if (!networks || !Array.isArray(networks) || networks.length === 0) return;
    
    networks.forEach((net, index) => {
        const card = document.createElement('div');
        card.className = 'network-card';
        card.innerHTML = `
            <div class="network-card-header">
                <span>Network Device ${index + 1}</span>
                <button type="button" class="btn-icon" onclick="this.closest('.network-card').remove()">&times;</button>
            </div>
            <div class="network-card-body">
                <div class="form-row">
                    <div class="form-group">
                        <label>Device</label>
                        <input type="text" class="form-input network-device" value="${net.device || ''}">
                    </div>
                    <div class="form-group">
                        <label>Boot Protocol</label>
                        <select class="form-select network-bootproto">
                            <option value="dhcp" ${net.bootProto === 'dhcp' ? 'selected' : ''}>DHCP</option>
                            <option value="static" ${net.bootProto === 'static' ? 'selected' : ''}>Static</option>
                        </select>
                    </div>
                </div>
                <div class="form-row">
                    <div class="form-group">
                        <label>IP Address</label>
                        <input type="text" class="form-input network-ip" placeholder="192.168.1.100" value="${net.ip || ''}">
                    </div>
                    <div class="form-group">
                        <label>Netmask</label>
                        <input type="text" class="form-input network-netmask" placeholder="255.255.255.0" value="${net.netmask || ''}">
                    </div>
                </div>
                <div class="form-row">
                    <div class="form-group">
                        <label>Gateway</label>
                        <input type="text" class="form-input network-gateway" placeholder="192.168.1.1" value="${net.gateway || ''}">
                    </div>
                    <div class="form-group">
                        <label>Hostname</label>
                        <input type="text" class="form-input network-hostname" placeholder="myhost" value="${net.hostname || ''}">
                    </div>
                </div>
                <div class="form-group">
                    <label class="checkbox-label">
                        <input type="checkbox" class="network-onboot" ${net.onBoot ? 'checked' : ''}>
                        Activate on boot
                    </label>
                </div>
            </div>
        `;
        networkDevices.appendChild(card);
    });
}

// Validation

/**
 * Validate the basic required fields that are marked with a red asterisk.
 * Highlights empty fields in red and shows an error status.
 *
 * Returns true if all required fields are filled; false otherwise.
 *
 * Required fields (must have a non-empty value):
 *   - systemHostname   (label has red asterisk)
 *   - rootPassword     (label has red asterisk)
 *
 * Checks at least one network device is present (net-device inputs with
 * a non-empty value) to match the backend's "network device is required"
 * validation.
 *
 * @param {string} statusId - DOM id of the status element to write to
 * @returns {boolean}
 */
function validateBasicRequired(statusId) {
    const requiredIds = [
        { id: 'systemHostname', msg: 'Hostname is required' },
        { id: 'rootPassword',   msg: 'Root password is required' },
    ];

    // Clear previous error highlights
    requiredIds.forEach(({ id }) => {
        const el = document.getElementById(id);
        if (el) el.classList.remove('input-error');
    });

    // Check network devices: at least one .net-device must be non-empty
    const netDevices = Array.from(document.querySelectorAll('.net-device'));
    netDevices.forEach(el => el.classList.remove('input-error'));
    const hasNetDevice = netDevices.some(el => el.value && el.value.trim() !== '');

    const errors = [];

    requiredIds.forEach(({ id, msg }) => {
        const el = document.getElementById(id);
        if (!el) return;
        if (!el.value || el.value.trim() === '') {
            el.classList.add('input-error');
            errors.push(msg);
        }
    });

    if (!hasNetDevice) {
        // Highlight all network devices red, since none are filled
        netDevices.forEach(el => el.classList.add('input-error'));
        errors.push('At least one network device is required');
    }

    if (errors.length > 0) {
        // Scroll to first error field: required fields first, then network devices
        const firstRequired = requiredIds.find(({ id }) => {
            const el = document.getElementById(id);
            return el && el.classList.contains('input-error');
        });
        if (firstRequired) {
            const el = document.getElementById(firstRequired.id);
            if (el) el.scrollIntoView({ behavior: 'smooth', block: 'center' });
        } else if (!hasNetDevice && netDevices.length > 0) {
            // Fall back to the first network device input
            netDevices[0].scrollIntoView({ behavior: 'smooth', block: 'center' });
        }
        // Show toast notification only (no inline status block)
        showToast(errors.join(', '), 'error');
        return false;
    }

    return true;
}

async function validateConfig() {
    console.log('=== validateConfig called ===');
    // Front-end required field gate — uses configActionStatus which is present
    // on the Configuration page (shared by all tabs).
    if (!validateBasicRequired('configActionStatus')) {
        return;
    }
    try {
        const config = getConfigFromForm();
        console.log('Config to validate:', JSON.stringify(config, null, 2));

        const response = await fetch(`${API_BASE}/config/validate`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(config)
        });

        console.log('Validate response status:', response.status);
        const data = await response.json();
        console.log('Validate response data:', data);

        if (data.success) {
            AppState.errors = data.data.errors || [];
            AppState.warnings = data.data.warnings || [];
            AppState.validationStatus = data.data.valid ? 'valid' : 'invalid';

            if (AppState.validationStatus === 'valid') {
                const validMsg = 'Config validation passed' + (AppState.warnings.length > 0 ? ` (${AppState.warnings.length} warning(s))` : '');
                showToast(validMsg, 'success');
            } else {
                const errMsg = `${AppState.errors.length} error(s) found`;
                showToast(errMsg, 'error');
            }
        } else {
            showToast('Validation failed: ' + (data.error || 'Unknown error'), 'error');
        }
    } catch (error) {
        console.error('Validation error:', error);
        showToast('Validation failed: ' + error.message, 'error');
    }
}

// Preview Generation
async function previewKickstart() {
    console.log('=== previewKickstart called ===');
    // Front-end required field gate — configActionStatus is the status element
    // in the Configuration page (shared by all tabs). Using 'ksConfigContent'
    // would match the ISO Builder's textarea instead of the active page's status.
    if (!validateBasicRequired('configActionStatus')) {
        return;
    }
    try {
        const config = getConfigFromForm();
        console.log('Config to preview:', JSON.stringify(config, null, 2));

        const response = await fetch(`${API_BASE}/config/generate`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(config)
        });

        console.log('Preview response status:', response.status);
        if (!response.ok) {
            const text = await response.text();
            console.error('Preview error response:', text);
            showToast('Failed to generate preview: HTTP ' + response.status, 'error');
            return;
        }

        const data = await response.json();
        console.log('Preview response data:', data);

        if (data.success) {
            // Switch to preview page
            if (window.AppNavigation && typeof window.AppNavigation.switchPage === 'function') {
                window.AppNavigation.switchPage('preview');
            } else if (typeof AppNavigation !== 'undefined') {
                AppNavigation.switchPage('preview');
            }

            // Update preview content
            const previewEl = document.getElementById('configPreview');

            if (previewEl) {
                previewEl.style.display = 'block';
                previewEl.textContent = data.data.config || '# No configuration generated';
            }
            showToast('Configuration preview generated', 'success');
        } else {
            showToast('Failed to generate preview: ' + (data.error || 'Unknown error'), 'error');
            console.error('Preview error:', data.error);
        }

        // Show the generate kickstart button
        const kickstartActions = document.getElementById('kickstartActions');
        if (kickstartActions) {
            kickstartActions.style.display = 'block';
        }
    } catch (error) {
        console.error('Failed to generate preview:', error);
        showToast('Failed to generate preview: ' + error.message, 'error');
    }
}

// Generate Kickstart to ISO Builder
function downloadKickstartConfig() {
    const previewEl = document.getElementById('configPreview');
    if (!previewEl) {
        showToast('No preview available', 'error');
        return;
    }

    const content = previewEl.textContent;
    if (!content) {
        showToast('No configuration to generate', 'error');
        return;
    }

    // Fill the Kickstart Configuration textarea in ISO Builder page
    const ksConfigContent = document.getElementById('ksConfigContent');
    if (ksConfigContent) {
        ksConfigContent.value = content;
    }

    // Switch to ISO Builder page
    AppNavigation.switchPage('iso');
    showToast('Kickstart configuration generated', 'success');
}

// Load Default Config
// Fetches the contents of templates/presets/default.cfg (or user/default.cfg
// if present) via /api/config/default, parses the kickstart string, and
// updates the configuration form. Used both by the "Load Default Config"
// button on the Configuration page and by the app-startup auto-load.
let defaultConfigLoadInFlight = false;

async function loadDefaultConfig() {
    if (defaultConfigLoadInFlight) {
        // Prevent a duplicate request (and a duplicate toast) when the user
        // clicks the button while the startup auto-load is still in flight.
        console.log('[loadDefaultConfig] request already in flight, skipping');
        return;
    }
    defaultConfigLoadInFlight = true;

    try {
        const response = await fetch(`${API_BASE}/config/default`);
        let data;
        try {
            data = await response.json();
        } catch (parseErr) {
            const msg = `Failed to load default configuration (HTTP ${response.status})`;
            showToast(msg, 'error');
            return;
        }

        if (!response.ok || !data.success || !data.config) {
            const msg = data.error || 'default.cfg is missing or invalid';
            showToast(msg, 'error');
            return;
        }

        // Show storage page actions after loading
        const storagePageActions = document.getElementById('storagePageActions');
        if (storagePageActions) {
            storagePageActions.style.display = 'block';
        }

        // Parse the kickstart config and update form
        const parseResponse = await fetch(`${API_BASE}/config/parse`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ config: data.config })
        });
        const parseResult = await parseResponse.json();

        if (parseResult.success && parseResult.data) {
            updateFormFromConfig(parseResult.data);
            const okMsg = `Default configuration (default.cfg) loaded successfully${data.name ? ` [${data.name}]` : ''}`;
            showToast(okMsg, 'success');
        } else {
            const errMsg = parseResult.error || 'Failed to parse default configuration';
            showToast(errMsg, 'error');
        }
    } catch (error) {
        console.error('Failed to load default config:', error);
        const msg = 'Failed to load default config: ' + error.message;
        showToast(msg, 'error');
    } finally {
        defaultConfigLoadInFlight = false;
    }
}

// Auto-load the default.cfg template on startup. Wraps loadDefaultConfig so
// the success/failure toast is always shown — the user wants explicit
// feedback whether the startup load worked or not.
async function autoLoadDefaultConfig() {
    try {
        await loadDefaultConfig();
    } catch (err) {
        // loadDefaultConfig already reports errors via toast; this is a
        // safety net so a thrown exception never breaks startup.
        console.error('[autoLoadDefaultConfig] unexpected error:', err);
        showToast('Failed to load default configuration on startup', 'error');
    }
}

// Distro Selection Handler
function initDistroHandler() {
    const select = document.getElementById('distroSelect');
    if (select) {
        select.addEventListener('change', () => {
            AppState.distro = select.value;
            AppState.isDirty = true;
            showToast(`Selected: ${select.options[select.selectedIndex].text}`, 'info', 2000);
        });
    }
}

// ISO Source Type Toggle
function toggleISOSourceType() {
    const sourceType = document.querySelector('input[name="isoSourceType"]:checked')?.value;
    const localSection = document.getElementById('localISOSection');
    const downloadSection = document.getElementById('downloadISOSection');

    if (sourceType === 'local') {
        if (localSection) localSection.style.display = 'block';
        if (downloadSection) downloadSection.style.display = 'none';
    } else if (sourceType === 'download') {
        if (localSection) localSection.style.display = 'none';
        if (downloadSection) downloadSection.style.display = 'block';
    }
}

// Install Media Type Toggle
// Currently a placeholder for any UI that depends on the cdrom/harddrive
// choice. The value is read directly from the radio group when the
// generate request is built.
function toggleInstallMediaType() {
    const mediaType = document.querySelector('input[name="installMediaType"]:checked')?.value;
    console.log('[InstallMedia] selected:', mediaType);
}

// Installation Source Type Toggle
function toggleInstallSourceOptions() {
    const sourceType = document.getElementById('installSourceType')?.value;
    const nfsOptions = document.getElementById('nfsOptions');
    const urlOptions = document.getElementById('urlOptions');
    const harddriveOptions = document.getElementById('harddriveOptions');

    // Hide all options first
    if (nfsOptions) nfsOptions.style.display = 'none';
    if (urlOptions) urlOptions.style.display = 'none';
    if (harddriveOptions) harddriveOptions.style.display = 'none';

    // Show the relevant options
    if (sourceType === 'nfs' && nfsOptions) {
        nfsOptions.style.display = 'block';
    } else if (sourceType === 'url' && urlOptions) {
        urlOptions.style.display = 'block';
    } else if (sourceType === 'harddrive' && harddriveOptions) {
        harddriveOptions.style.display = 'block';
    }
}

// ISO File Selection — upload to server with progress (matches UbuntuCraft)
// Set Generate ISO Image button disabled/enabled state. When disabled,
// the btn-disabled class is added so the button is visually greyed out
// (see base.css .btn.btn-disabled). Matches the UbuntuCraft pattern.
function setGenerateButtonDisabled(disabled, reasonText) {
    const generateBtn = document.getElementById('generateBtn');
    if (!generateBtn) return;
    generateBtn.disabled = !!disabled;
    if (disabled) {
        generateBtn.classList.add('btn-disabled');
        if (reasonText) generateBtn.textContent = reasonText;
    } else {
        generateBtn.classList.remove('btn-disabled');
        generateBtn.textContent = 'Generate ISO Image';
    }
}

function handleISOFileSelect(input) {
    const file = input.files[0];
    if (!file) return;

    const fileInfo = document.getElementById('isoFileInfo');
    const fileName = document.getElementById('isoFileName');
    const fileSize = document.getElementById('isoFileSize');
    const progressContainer = document.getElementById('uploadProgressContainer');
    const progressFill = document.getElementById('uploadProgressFill');
    const progressText = document.getElementById('uploadProgressText');
    const uploadStatusText = document.getElementById('uploadStatusText');
    const sourceISOInput = document.getElementById('sourceISO');

    if (fileInfo) fileInfo.style.display = 'flex';
    if (fileName) fileName.textContent = file.name;
    if (fileSize) fileSize.textContent = formatFileSize(file.size);

    if (progressContainer) progressContainer.style.display = 'block';
    if (progressFill) progressFill.style.width = '0%';
    if (progressText) progressText.textContent = '0%';
    if (uploadStatusText) {
        uploadStatusText.textContent = 'Uploading...';
        uploadStatusText.classList.remove('status-success', 'status-error');
    }
    if (sourceISOInput) sourceISOInput.value = '';

    // Disable Generate ISO Image button while the ISO is uploading so
    // the user cannot trigger a build against a half-uploaded source.
    setGenerateButtonDisabled(true, 'Uploading ISO...');

    const apiBase = (typeof window.API_BASE === 'string' && window.API_BASE)
        ? window.API_BASE
        : (window.location && window.location.origin ? window.location.origin : '');

    const xhr = new XMLHttpRequest();
    const formData = new FormData();
    formData.append('iso', file);

    xhr.upload.onprogress = function (e) {
        if (!e.lengthComputable) return;
        const percent = Math.round((e.loaded / e.total) * 100);
        if (progressFill) progressFill.style.width = percent + '%';
        if (progressText) progressText.textContent = percent + '%';
    };

    xhr.onload = function () {
        let resp = null;
        try { resp = JSON.parse(xhr.responseText); } catch (err) { /* ignore */ }

        if (xhr.status >= 200 && xhr.status < 300 && resp && resp.success) {
            if (progressFill) progressFill.style.width = '100%';
            if (progressText) progressText.textContent = '100%';
            if (uploadStatusText) {
                uploadStatusText.textContent = 'Upload completed';
                uploadStatusText.classList.add('status-success');
            }
            if (sourceISOInput && resp.filePath) {
                sourceISOInput.value = resp.filePath;
            }
            // Upload finished — re-enable Generate ISO button so the
            // user can now build against the just-uploaded source.
            setGenerateButtonDisabled(false);
            showToast(`Uploaded: ${file.name}`, 'success', 2000);
        } else {
            const msg = (resp && resp.error) ? resp.error : `Upload failed (HTTP ${xhr.status})`;
            if (uploadStatusText) {
                uploadStatusText.textContent = 'Upload error: ' + msg;
                uploadStatusText.classList.add('status-error');
            }
            setGenerateButtonDisabled(false);
            showToast(msg, 'error', 3000);
        }
    };

    xhr.onerror = function () {
        if (uploadStatusText) {
            uploadStatusText.textContent = 'Upload error: network error';
            uploadStatusText.classList.add('status-error');
        }
        setGenerateButtonDisabled(false);
        window.showToast('Network error during upload', 'error', 3000);
    };

    xhr.onabort = function () {
        if (uploadStatusText) {
            uploadStatusText.textContent = 'Upload cancelled (computer slept or network lost)';
            uploadStatusText.classList.add('status-error');
        }
        setGenerateButtonDisabled(false);
        window.showToast('Upload cancelled — please retry', 'warn', 4000);
    };

    // Warn user if they switch tabs or the window loses focus while uploading
    // (common on macOS when closing the laptop lid).
    let wasUploading = true;
    const onVisibilityChange = function () {
        if (document.visibilityState === 'hidden' && wasUploading) {
            wasUploading = false;
            if (uploadStatusText) {
                uploadStatusText.textContent = 'Uploading... (do not close this tab)';
            }
        } else if (document.visibilityState === 'visible') {
            wasUploading = true;
        }
    };
    document.addEventListener('visibilitychange', onVisibilityChange);

    xhr.open('POST', apiBase + '/api/iso/upload', true);
    xhr.send(formData);
}

function removeSelectedISO() {
    const input = document.getElementById('isoFileInput');
    const fileInfo = document.getElementById('isoFileInfo');
    const progressContainer = document.getElementById('uploadProgressContainer');
    const progressFill = document.getElementById('uploadProgressFill');
    const progressText = document.getElementById('uploadProgressText');
    const uploadStatusText = document.getElementById('uploadStatusText');
    const sourceISOInput = document.getElementById('sourceISO');

    if (input) input.value = '';
    if (fileInfo) fileInfo.style.display = 'none';
    if (progressContainer) progressContainer.style.display = 'none';
    if (progressFill) progressFill.style.width = '0%';
    if (progressText) progressText.textContent = '0%';
    if (uploadStatusText) {
        uploadStatusText.textContent = 'Uploading...';
        uploadStatusText.classList.remove('status-success', 'status-error');
    }
    if (sourceISOInput) sourceISOInput.value = '';
    // Source ISO was cleared, but for local mode the user has not
    // uploaded anything yet — keep the button enabled so the form is
    // not stranded, but the validator in ISOGenerator.generate() will
    // catch a missing source. Reset to plain "Generate ISO Image".
    setGenerateButtonDisabled(false);
}

function formatFileSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
    return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB';
}

// Reset Build Config — show a confirmation modal first (matches
// UbuntuCraft's resetBuildConfig). Only after the user clicks Confirm
// do we wipe the build state.
function resetBuildConfig() {
    console.log('[resetBuildConfig] Function called');
    showConfirmModal(
        'Reset Configuration',
        'Reset all build configuration? This will clear source selection, destination path, progress state, and the build directory (build/mnt/{packages,script}).',
        performResetBuildConfig
    );
}

function performResetBuildConfig() {
    // Call backend to wipe build/mnt/* and rebuild skeleton
    fetch('/api/iso/reset', { method: 'POST' })
        .then(res => res.json())
        .then(data => {
            if (!data.success) {
                window.showToast('Reset failed: ' + (data.error || 'unknown error'), 'error');
                return;
            }
        })
        .catch(err => window.showToast('Reset request failed: ' + err.message, 'error'));

    // --- Frontend cleanup (runs immediately, backend cleanup is async) ---
    const destinationISO = document.getElementById('destinationISO');
    const ksConfigContent = document.getElementById('ksConfigContent');
    const isoStatus = document.getElementById('isoStatus');
    const buildProgress = document.getElementById('buildProgress');
    const buildLogs = document.getElementById('buildLogs');
    const downloadSection = document.getElementById('downloadSection');
    const progressFill = document.getElementById('progressFill');
    const progressText = document.getElementById('progressText');
    const logContainer = document.getElementById('logContainer');

    // Also clear any in-progress upload widget state
    const isoFileInput = document.getElementById('isoFileInput');
    const isoFileInfo = document.getElementById('isoFileInfo');
    const uploadProgressContainer = document.getElementById('uploadProgressContainer');
    const uploadStatusText = document.getElementById('uploadStatusText');
    const sourceISOInput = document.getElementById('sourceISO');

    if (destinationISO) destinationISO.value = 'custom-kickstart.iso';
    if (ksConfigContent) ksConfigContent.value = '';
    if (isoFileInput) isoFileInput.value = '';
    if (isoFileInfo) isoFileInfo.style.display = 'none';
    if (uploadProgressContainer) uploadProgressContainer.style.display = 'none';
    if (uploadStatusText) {
        uploadStatusText.textContent = 'Uploading...';
        uploadStatusText.classList.remove('status-success', 'status-error');
    }
    if (sourceISOInput) sourceISOInput.value = '';
    if (isoStatus) {
        isoStatus.style.display = 'none';
        isoStatus.className = 'status';
        isoStatus.textContent = '';
    }
    if (buildProgress) buildProgress.style.display = 'none';
    if (buildLogs) buildLogs.style.display = 'none';
    if (downloadSection) downloadSection.style.display = 'none';
    if (progressFill) progressFill.style.width = '0%';
    if (progressText) progressText.textContent = '';
    if (logContainer) logContainer.innerHTML = '';

    // Re-enable the Generate button and restore its label
    setGenerateButtonDisabled(false);

    // Stop any in-progress polling and reset ISOGenerator state
    if (window.isoGenerator) {
        window.isoGenerator.stopPolling();
        window.isoGenerator.currentTask = null;
        window.isoGenerator.logOffset = 0;
    }

    showToast('Build config reset', 'info', 2000);
}

// Ignoredisk UI Update
function updateIgnorediskUI() {
    const mode = document.querySelector('input[name="ignorediskMode"]:checked')?.value;
    const drivesSection = document.getElementById('ignorediskDrivesSection');
    const onlyuseSection = document.getElementById('onlyuseDrivesSection');

    if (drivesSection) {
        drivesSection.style.display = mode === 'ignore' ? 'block' : 'none';
    }
    if (onlyuseSection) {
        onlyuseSection.style.display = mode === 'onlyuse' ? 'block' : 'none';
    }
}

// Partition Mode Toggle
function togglePartitionMode() {
    const mode = document.querySelector('input[name="partitionMode"]:checked')?.value;
    const autoSection = document.getElementById('autopartSection');
    const manualSection = document.getElementById('manualPartitionSection');

    if (mode === 'auto') {
        if (autoSection) autoSection.style.display = 'block';
        if (manualSection) manualSection.style.display = 'none';
    } else {
        if (autoSection) autoSection.style.display = 'none';
        if (manualSection) manualSection.style.display = 'block';
    }
}

// Initialize Application
async function initApp() {
    console.log('KickCraft initializing...');

    // Initialize config manager first (other managers depend on it)
    if (typeof ConfigManager !== 'undefined') {
        window.configManager = new ConfigManager();
        console.log('ConfigManager initialized');
    }

    // Initialize tabs
    initTabs();

    // Initialize distro handler
    initDistroHandler();

    // Initialize storage manager
    if (typeof StorageManager !== 'undefined') {
        window.StorageManager = new StorageManager();
    }

    // Initialize network manager
    if (typeof NetworkManager !== 'undefined') {
        window.networkManager = new NetworkManager();
    }

    // Initialize system info manager (it's already an object, not a class)
    if (typeof SystemInfoManager !== 'undefined') {
        window.SystemInfoManager = SystemInfoManager;
    }

    // Initialize ISO generator
    if (typeof ISOGenerator !== 'undefined') {
        window.isoGenerator = new ISOGenerator();
    }

    // Initialize templates manager
    console.log('[App] Checking TemplatesManager:', typeof TemplatesManager);
    if (typeof TemplatesManager !== 'undefined') {
        window.templatesManager = new TemplatesManager();
        console.log('[App] TemplatesManager instantiated');
        await window.templatesManager.loadTemplates();
        console.log('[App] TemplatesManager.loadTemplates completed');
    } else {
        console.error('[App] TemplatesManager class not found!');
    }

    // Load data
    await loadDistros();

    // Initialize repo list with one default empty entry
    loadReposFromConfig([]);

    // Setup event listeners
    setupEventListeners();

    // Initialize system-info page — it is the default active page,
    // so its data must be loaded on startup, not only when the user
    // clicks the sidebar nav (otherwise the page sits on "Loading..."
    // until the user navigates away and back).
    if (typeof SystemInfoManager !== 'undefined') {
        SystemInfoManager.init();
    }

    // Auto-load the default.cfg template so the configuration page is
    // populated as soon as the app is ready. The user gets a success or
    // failure toast notification either way.
    await autoLoadDefaultConfig();

    console.log('KickCraft initialized');
    console.log('[App] templatesManager on window:', window.templatesManager);
}

function setupEventListeners() {
    // Validate button
    const validateBtn = document.getElementById('validateConfigBtn');
    if (validateBtn) {
        validateBtn.addEventListener('click', validateConfig);
    }

    // Ignoredisk checkbox
    const enableIgnoredisk = document.getElementById('enableIgnoredisk');
    if (enableIgnoredisk) {
        enableIgnoredisk.addEventListener('change', function() {
            const options = document.getElementById('ignorediskOptions');
            if (options) {
                options.style.display = this.checked ? 'block' : 'none';
            }
        });
    }

    // Health check
    const healthBtn = document.getElementById('healthBtn');
    if (healthBtn) {
        healthBtn.addEventListener('click', checkHealth);
    }

    // ISO source type toggle
    const isoSourceRadios = document.querySelectorAll('input[name="isoSourceType"]');
    isoSourceRadios.forEach(radio => {
        radio.addEventListener('change', toggleISOSourceType);
    });

    // Install media type toggle
    const installMediaRadios = document.querySelectorAll('input[name="installMediaType"]');
    installMediaRadios.forEach(radio => {
        radio.addEventListener('change', toggleInstallMediaType);
    });

    // Add User button
    const addUserBtn = document.getElementById('addUserBtn');
    if (addUserBtn) {
        addUserBtn.addEventListener('click', addUserRow);
    }

    // Add Repo button
    const addRepoBtn = document.getElementById('addRepoBtn');
    console.log('[setupEventListeners] addRepoBtn:', addRepoBtn);
    if (addRepoBtn) {
        addRepoBtn.addEventListener('click', addRepoFromForm);
        console.log('[setupEventListeners] Add Repo click listener attached');
    } else {
        console.error('[setupEventListeners] addRepoBtn not found!');
    }

    // Add Repo on Enter key
    const newRepoName = document.getElementById('newRepoName');
    const newRepoBaseUrl = document.getElementById('newRepoBaseUrl');
    if (newRepoName) {
        newRepoName.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                e.preventDefault();
                if (newRepoBaseUrl) newRepoBaseUrl.focus();
            }
        });
    }
    if (newRepoBaseUrl) {
        newRepoBaseUrl.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                e.preventDefault();
                addRepoFromForm();
            }
        });
    }
}

// Add User Row
function addUserRow() {
    const usersList = document.getElementById('usersList');
    const template = document.getElementById('userItemTemplate');
    if (!usersList) return;

    let userItem;
    if (template) {
        const clone = template.content.cloneNode(true);
        userItem = clone.querySelector('.user-item');
    } else {
        userItem = document.createElement('div');
        userItem.className = 'user-item';
        userItem.innerHTML = createUserItemHTML();
    }

    usersList.appendChild(userItem);

    // Bind delete button
    userItem.querySelector('.user-delete').addEventListener('click', () => {
        userItem.remove();
    });
}

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', initApp);

// Export for use in other modules
window.AppState = AppState;
window.AppNavigation = AppNavigation;
window.showToast = showToast;
window.showConfirmModal = showConfirmModal;
window.openModal = openModal;
window.closeModal = closeModal;
window.getConfigFromForm = getConfigFromForm;
window.validateBasicRequired = validateBasicRequired;
window.updateFormFromConfig = updateFormFromConfig;
window.previewKickstart = previewKickstart;
window.loadDefaultConfig = loadDefaultConfig;
window.autoLoadDefaultConfig = autoLoadDefaultConfig;
window.validateConfig = validateConfig;
window.showStatus = showStatus;
window.toggleISOSourceType = toggleISOSourceType;
window.toggleInstallSourceOptions = toggleInstallSourceOptions;
window.toggleInstallMediaType = toggleInstallMediaType;
window.setGenerateButtonDisabled = setGenerateButtonDisabled;
window.handleISOFileSelect = handleISOFileSelect;
window.removeSelectedISO = removeSelectedISO;
window.resetBuildConfig = resetBuildConfig;
window.updateIgnorediskUI = updateIgnorediskUI;
window.addUserRow = addUserRow;

// ============ Repo Management ============

function loadReposFromConfig(repos) {
    const repoList = document.getElementById('repoList');
    if (!repoList) return;

    repoList.innerHTML = '';

    if (!repos || repos.length === 0) {
        return;
    }

    repos.forEach(repo => {
        addRepoItem(repo.name, repo.baseurl || repo.BaseURL || '');
    });
}

function addRepoItem(name, url) {
    const repoList = document.getElementById('repoList');
    if (!repoList) return;

    const repoCard = document.createElement('div');
    repoCard.className = 'repo-card';
    repoCard.innerHTML = `
        <div class="repo-card-header">
            <span class="repo-card-type">Repository</span>
            <button type="button" class="remove-btn" onclick="removeRepoCard(this)">Remove</button>
        </div>
        <div class="form-row">
            <div class="form-group">
                <label class="optional">Name <span class="hint-icon" data-tooltip="Repository display name (optional)">?</span></label>
                <input type="text" class="repo-name-input" value="${escapeHtml(name)}" placeholder="e.g., AppStream">
            </div>
            <div class="form-group">
                <label class="optional">Base URL <span class="hint-icon" data-tooltip="Repository base URL">?</span></label>
                <input type="text" class="repo-url-input" value="${escapeHtml(url)}" placeholder="e.g., file:///run/install/repo/">
            </div>
        </div>
    `;

    repoList.appendChild(repoCard);
}

function removeRepoCard(btn) {
    btn.closest('.repo-card').remove();
}

function addRepoFromForm() {
    // Add a new blank repo card
    addRepoItem('', '');
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
