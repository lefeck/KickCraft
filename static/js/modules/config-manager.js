// Configuration Manager Module
// Handles configuration state and synchronization

class ConfigManager {
    constructor() {
        this.config = this.getDefaultConfig();
        this.listeners = [];
    }

    getDefaultConfig() {
        return {
            locale: {
                lang: 'en_US.UTF-8',
                keymap: 'us',
                xlayouts: '',
                addSupport: '',
                timezone: 'UTC',
                utc: true,
                noNtp: false,
                ntpServers: ''
            },
            rootPassword: {
                password: '',
                isCrypted: false,
                lock: true,
                isSet: false
            },
            users: [],
            bootloader: {
                location: 'mbr',
                append: '',
                bootDrive: '',
                driveOrder: []
            },
            storage: {
                zerombr: false,
                clearAll: false,
                clearLinux: false,
                clearDrives: [],
                initLabel: false,
                ignoreDiskDrives: [],
                partitions: [],
                raids: [],
                volGroups: [],
                logVols: []
            },
            network: [],
            firewall: {
                enabled: true,
                services: ['ssh'],
                ports: [],
                trust: []
            },
            selinux: {
                mode: 'enforcing'
            },
            auth: {
                enableShadow: true,
                passwordAlgorithm: 'sha512'
            },
            services: {
                enabled: [],
                disabled: ['chronyd', 'postfix']
            },
            kdump: {
                enabled: true,
                reserveMb: 'auto'
            },
            repos: [],
            packages: {
                packages: [],
                groups: [],
                languages: [],
                default: false,
                noBase: false,
                excludeDocs: false,
                ignoreMissing: false
            },
            preScripts: [],
            postScripts: [],
            postNoChrootScripts: [],
            graphics: {
                skipX: false,
                firstBoot: 'enabled'
            },
            eula: ''
        };
    }

    // State management
    getConfig() {
        return { ...this.config };
    }

    setConfig(config) {
        this.config = { ...config };
        this.notifyListeners();
    }

    resetConfig() {
        this.config = this.getDefaultConfig();
        this.notifyListeners();
    }

    // Listeners
    addListener(callback) {
        this.listeners.push(callback);
    }

    removeListener(callback) {
        const index = this.listeners.indexOf(callback);
        if (index > -1) {
            this.listeners.splice(index, 1);
        }
    }

    notifyListeners() {
        this.listeners.forEach(callback => callback(this.config));
    }

    // Locale
    setLocale(locale) {
        Object.assign(this.config.locale, locale);
        this.notifyListeners();
    }

    // Root Password
    setRootPassword(password, options = {}) {
        this.config.rootPassword.password = password;
        this.config.rootPassword.isSet = !!password;
        Object.assign(this.config.rootPassword, options);
        this.notifyListeners();
    }

    // Bootloader
    setBootloader(bootloader) {
        Object.assign(this.config.bootloader, bootloader);
        this.notifyListeners();
    }

    // Storage
    addPartition(partition) {
        this.config.storage.partitions.push(partition);
        this.notifyListeners();
    }

    removePartition(index) {
        this.config.storage.partitions.splice(index, 1);
        this.notifyListeners();
    }

    updatePartition(index, partition) {
        if (index >= 0 && index < this.config.storage.partitions.length) {
            Object.assign(this.config.storage.partitions[index], partition);
            this.notifyListeners();
        }
    }

    addVolGroup(volGroup) {
        this.config.storage.volGroups.push(volGroup);
        this.notifyListeners();
    }

    removeVolGroup(index) {
        this.config.storage.volGroups.splice(index, 1);
        this.notifyListeners();
    }

    addLogVol(logVol) {
        this.config.storage.logVols.push(logVol);
        this.notifyListeners();
    }

    removeLogVol(index) {
        this.config.storage.logVols.splice(index, 1);
        this.notifyListeners();
    }

    // Network
    addNetwork(network) {
        this.config.network.push(network);
        this.notifyListeners();
    }

    removeNetwork(index) {
        this.config.network.splice(index, 1);
        this.notifyListeners();
    }

    updateNetwork(index, network) {
        if (index >= 0 && index < this.config.network.length) {
            Object.assign(this.config.network[index], network);
            this.notifyListeners();
        }
    }

    // Firewall
    setFirewall(firewall) {
        Object.assign(this.config.firewall, firewall);
        this.notifyListeners();
    }

    // SELinux
    setSELinux(mode) {
        this.config.selinux.mode = mode;
        this.notifyListeners();
    }

    // Services
    setServices(services) {
        Object.assign(this.config.services, services);
        this.notifyListeners();
    }

    // Packages
    addPackage(packageName) {
        if (!this.config.packages.packages.includes(packageName)) {
            this.config.packages.packages.push(packageName);
            this.notifyListeners();
        }
    }

    removePackage(packageName) {
        const index = this.config.packages.packages.indexOf(packageName);
        if (index > -1) {
            this.config.packages.packages.splice(index, 1);
            this.notifyListeners();
        }
    }

    addGroup(groupName) {
        const fullName = groupName.startsWith('@^') ? groupName : `@^${groupName}`;
        if (!this.config.packages.groups.some(g => g.name === fullName)) {
            this.config.packages.groups.push({ name: fullName });
            this.notifyListeners();
        }
    }

    removeGroup(groupName) {
        const index = this.config.packages.groups.findIndex(g => g.name === groupName || g.name === `@^${groupName}`);
        if (index > -1) {
            this.config.packages.groups.splice(index, 1);
            this.notifyListeners();
        }
    }

    // Scripts
    setPreScript(content, options = {}) {
        this.config.preScripts = [{
            type: 'pre',
            content: content,
            interpreter: options.interpreter || '/bin/bash',
            noChroot: true,
            errorOnFail: options.errorOnFail || false
        }];
        this.notifyListeners();
    }

    setPostScript(content, options = {}) {
        this.config.postScripts = [{
            type: 'post',
            content: content,
            interpreter: options.interpreter || '/bin/bash',
            noChroot: options.noChroot || false,
            errorOnFail: options.errorOnFail || false
        }];
        this.notifyListeners();
    }

    setPostNoChrootScript(content, options = {}) {
        this.config.postNoChrootScripts = [{
            type: 'post',
            content: content,
            interpreter: options.interpreter || '/bin/bash',
            noChroot: true,
            errorOnFail: options.errorOnFail || false
        }];
        this.notifyListeners();
    }

    // Validation
    validate() {
        const errors = [];
        const warnings = [];

        // Check locale
        if (!this.config.locale.lang) {
            errors.push('Language is required');
        }

        // Check storage
        if (!this.config.storage.zerombr && 
            !this.config.storage.clearAll && 
            this.config.storage.partitions.length === 0) {
            warnings.push('Consider adding zerombr or clearpart to avoid interactive partitioning');
        }

        // Check partitions
        this.config.storage.partitions.forEach((part, i) => {
            if (!part.mountpoint && !part.fstype) {
                errors.push(`Partition ${i + 1}: must specify mountpoint or fstype`);
            }
        });

        // Check root password
        if (!this.config.rootPassword.isSet) {
            warnings.push('Root password not set - installation may be interactive');
        }

        return { valid: errors.length === 0, errors, warnings };
    }

    // Serialization
    toJSON() {
        return JSON.stringify(this.config, null, 2);
    }

    fromJSON(json) {
        try {
            this.config = JSON.parse(json);
            this.notifyListeners();
            return true;
        } catch (e) {
            console.error('Failed to parse config JSON:', e);
            return false;
        }
    }

    // Generate ISO Image — delegates to ISOGenerator singleton
    async generateISO() {
        console.log('[ConfigManager.generateISO] called, window.ISOGenerator:', typeof window.ISOGenerator);
        try {
            if (window.ISOGenerator) {
                if (!window.isoGenerator) {
                    window.isoGenerator = new window.ISOGenerator();
                    console.log('[ConfigManager.generateISO] created isoGenerator instance');
                }
                return window.isoGenerator.generate();
            }
            console.error('[ConfigManager.generateISO] ISOGenerator not loaded on window');
            return null;
        } catch (err) {
            console.error('[ConfigManager.generateISO] uncaught error:', err);
            window.showToast('Generation error: ' + err.message, 'error');
        }
    }

    // Download the generated ISO
    downloadISO() {
        if (!window.isoGenerator || !window.isoGenerator.currentTask) {
            showToast('No ISO available for download', 'error');
            return;
        }
        window.isoGenerator.downloadCurrentISO();
    }
}

// Export for global use
window.ConfigManager = ConfigManager;
