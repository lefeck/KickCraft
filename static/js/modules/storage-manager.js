/**
 * Storage Management Module
 * Kickstart storage configuration with card-based UI style
 */

class StorageManager {
    constructor() {
        this.storageConfigs = null;
        this.init();
        this.initUI();
    }

    init() {
        this.storageConfigs = document.getElementById('storageConfigs');

        if (this.storageConfigs && this.storageConfigs.children.length === 0) {
            this.addDefaultPartitions();
        }
    }

    addDefaultPartitions() {
        const defaults = [
            { mountpoint: '/boot', fstype: 'xfs', size: 1024, grow: false, asPrimary: true, onDisk: '' },
            { mountpoint: '/', fstype: 'xfs', size: 10240, grow: true, asPrimary: false, onDisk: '' }
        ];

        defaults.forEach(part => {
            this.addStorageCard('partition', part);
        });
        this.syncStorageConfig();
    }

    setPartitionMode(mode) {
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

    getDefaultDataForType(type) {
        switch (type) {
            case 'partition':
                return {
                    mountpoint: '',
                    fstype: 'xfs',
                    size: 1024,
                    onDisk: '',
                    grow: false,
                    asPrimary: false,
                    encrypted: false
                };
            case 'volgroup':
                return {
                    name: 'vg00',
                    physicalVolumes: ['sda2'],
                    pesize: '4096'
                };
            case 'logvol':
                return {
                    vgname: 'vg00',
                    name: '',
                    mountpoint: '',
                    size: 4096,
                    fstype: 'xfs',
                    grow: false,
                    encrypted: false
                };
            case 'raid':
                return {
                    mountpoint: '',
                    level: '1',
                    device: 'md0',
                    fstype: 'xfs',
                    devices: ['sda1', 'sdb1'],
                    spares: 0,
                    encrypted: false
                };
            case 'btrfs':
                return {
                    subvol: '',
                    name: '',
                    devices: ['sda3'],
                    level: '',
                    metaLevel: '',
                    label: ''
                };
            default:
                return {};
        }
    }

    normalizeDataForType(type, data = {}) {
        return {
            ...this.getDefaultDataForType(type),
            ...data
        };
    }

    getStorageCardTitle(type) {
        switch (type) {
            case 'partition':
                return 'Partition Configuration';
            case 'volgroup':
                return 'Volume Group Configuration';
            case 'logvol':
                return 'Logical Volume Configuration';
            case 'raid':
                return 'RAID Configuration';
            case 'btrfs':
                return 'Btrfs Configuration';
            default:
                return 'Storage Configuration';
        }
    }

    getDiskTypeSelectorHTML(type) {
        return `
            <div class="form-row storage-card-type-row">
                <div class="form-group">
                    <label>Disk Type</label>
                    <select class="storage-card-type-select">
                        <option value="partition" ${type === 'partition' ? 'selected' : ''}>Partition</option>
                        <option value="volgroup" ${type === 'volgroup' ? 'selected' : ''}>Volume Groups</option>
                        <option value="logvol" ${type === 'logvol' ? 'selected' : ''}>Logical Volumes</option>
                        <option value="raid" ${type === 'raid' ? 'selected' : ''}>RAID</option>
                        <option value="btrfs" ${type === 'btrfs' ? 'selected' : ''}>Btrfs</option>
                    </select>
                </div>
                <div class="form-group storage-card-type-spacer" aria-hidden="true"></div>
            </div>
        `;
    }

    getPartitionFieldsHTML(partition = {}) {
        return `
            <div class="form-row">
                <div class="form-group">
                    <label>Disk Type</label>
                    <select class="storage-card-type-select">
                        <option value="partition" selected>Partition</option>
                        <option value="volgroup">Volume Groups</option>
                        <option value="logvol">Logical Volumes</option>
                        <option value="raid">RAID</option>
                        <option value="btrfs">Btrfs</option>
                    </select>
                </div>
                <div class="form-group">
                    <label class="no-required">Mount Point</label>
                    <input type="text" class="partition-mount" value="${partition.mountpoint || ''}" placeholder="/home">
                </div>
            </div>
            <div class="form-row">
                <div class="form-group">
                    <label>File System</label>
                    <select class="partition-fstype">
                        <option value="xfs" ${partition.fstype === 'xfs' ? 'selected' : ''}>XFS</option>
                        <option value="ext4" ${partition.fstype === 'ext4' ? 'selected' : ''}>Ext4</option>
                        <option value="ext3" ${partition.fstype === 'ext3' ? 'selected' : ''}>Ext3</option>
                        <option value="btrfs" ${partition.fstype === 'btrfs' ? 'selected' : ''}>Btrfs</option>
                        <option value="swap" ${partition.fstype === 'swap' ? 'selected' : ''}>Swap</option>
                    </select>
                </div>
                <div class="form-group">
                    <label>Disk</label>
                    <input type="text" class="partition-ondisk" value="${partition.onDisk || partition.ondisk || ''}" placeholder="vda">
                </div>
            </div>
            <div class="form-row">
                <div class="form-group">
                    <label>Size (MB)</label>
                    <input type="number" class="partition-size" value="${partition.size || 1024}" min="1">
                </div>
                <div class="form-group storage-form-spacer" aria-hidden="true"></div>
            </div>
            <div class="form-row storage-toggle-row">
                <div class="storage-toggle-item">
                    <label class="switch">
                        <input type="checkbox" class="partition-grow" ${partition.grow ? 'checked' : ''}>
                        <span class="slider"></span>
                    </label>
                    <span class="switch-label">Grow (use remaining space)</span>
                </div>
                <div class="storage-toggle-item">
                    <label class="switch">
                        <input type="checkbox" class="partition-primary" ${partition.asPrimary ? 'checked' : ''}>
                        <span class="slider"></span>
                    </label>
                    <span class="switch-label">Primary</span>
                </div>
                <div class="storage-toggle-item">
                    <label class="switch">
                        <input type="checkbox" class="partition-encrypted" ${partition.encrypted ? 'checked' : ''}>
                        <span class="slider"></span>
                    </label>
                    <span class="switch-label">Encrypt</span>
                </div>
            </div>
        `;
    }

    getVolGroupFieldsHTML(volGroup = {}) {
        return `
            <div class="form-row">
                <div class="form-group">
                    <label>Disk Type</label>
                    <select class="storage-card-type-select">
                        <option value="partition">Partition</option>
                        <option value="volgroup" selected>Volume Groups</option>
                        <option value="logvol">Logical Volumes</option>
                        <option value="raid">RAID</option>
                        <option value="btrfs">Btrfs</option>
                    </select>
                </div>
                <div class="form-group">
                    <label>Volume Group Name</label>
                    <input type="text" class="vg-name" value="${volGroup.name || 'vg00'}" placeholder="vg00">
                </div>
            </div>
            <div class="form-row">
                <div class="form-group">
                    <label>Physical Volumes (space separated)</label>
                    <input type="text" class="vg-pvs" value="${volGroup.physicalVolumes?.join(' ') || 'sda2'}" placeholder="sda2">
                </div>
                <div class="form-group">
                    <label>PE Size</label>
                    <select class="vg-pesize">
                        <option value="4096" ${volGroup.pesize === '4096' ? 'selected' : ''}>4 MiB</option>
                        <option value="8192" ${volGroup.pesize === '8192' ? 'selected' : ''}>8 MiB</option>
                        <option value="16384" ${volGroup.pesize === '16384' ? 'selected' : ''}>16 MiB</option>
                        <option value="32768" ${volGroup.pesize === '32768' ? 'selected' : ''}>32 MiB</option>
                    </select>
                </div>
            </div>
        `;
    }

    getLogVolFieldsHTML(logVol = {}) {
        return `
            <div class="form-row">
                <div class="form-group">
                    <label>Disk Type</label>
                    <select class="storage-card-type-select">
                        <option value="partition">Partition</option>
                        <option value="volgroup">Volume Groups</option>
                        <option value="logvol" selected>Logical Volumes</option>
                        <option value="raid">RAID</option>
                        <option value="btrfs">Btrfs</option>
                    </select>
                </div>
                <div class="form-group">
                    <label>VG Name</label>
                    <input type="text" class="lv-vgname" value="${logVol.vgname || 'vg00'}" placeholder="vg00">
                </div>
            </div>
            <div class="form-row">
                <div class="form-group">
                    <label>LV Name</label>
                    <input type="text" class="lv-name" value="${logVol.name || ''}" placeholder="lv_root">
                </div>
                <div class="form-group">
                    <label class="no-required">Mount Point</label>
                    <input type="text" class="lv-mount" value="${logVol.mountpoint || ''}" placeholder="/home">
                </div>
            </div>
            <div class="form-row">
                <div class="form-group">
                    <label>File System</label>
                    <select class="lv-fstype">
                        <option value="xfs" ${logVol.fstype === 'xfs' ? 'selected' : ''}>XFS</option>
                        <option value="ext4" ${logVol.fstype === 'ext4' ? 'selected' : ''}>Ext4</option>
                        <option value="btrfs" ${logVol.fstype === 'btrfs' ? 'selected' : ''}>Btrfs</option>
                        <option value="swap" ${logVol.fstype === 'swap' ? 'selected' : ''}>Swap</option>
                    </select>
                </div>
                <div class="form-group">
                    <label>Size (MB)</label>
                    <input type="number" class="lv-size" value="${logVol.size || 4096}" min="1">
                </div>
            </div>
            <div class="form-row storage-toggle-row">
                <div class="storage-toggle-item">
                    <label class="switch">
                        <input type="checkbox" class="lv-grow" ${logVol.grow ? 'checked' : ''}>
                        <span class="slider"></span>
                    </label>
                    <span class="switch-label">Grow (use remaining space)</span>
                </div>
                <div class="storage-toggle-item">
                    <label class="switch">
                        <input type="checkbox" class="lv-encrypted" ${logVol.encrypted ? 'checked' : ''}>
                        <span class="slider"></span>
                    </label>
                    <span class="switch-label">Encrypt</span>
                </div>
            </div>
        `;
    }

    getRaidFieldsHTML(raid = {}) {
        return `
            <div class="form-row">
                <div class="form-group">
                    <label>Disk Type</label>
                    <select class="storage-card-type-select">
                        <option value="partition">Partition</option>
                        <option value="volgroup">Volume Groups</option>
                        <option value="logvol">Logical Volumes</option>
                        <option value="raid" selected>RAID</option>
                        <option value="btrfs">Btrfs</option>
                    </select>
                </div>
                <div class="form-group">
                    <label class="no-required">Mount Point</label>
                    <input type="text" class="raid-mount" value="${raid.mountpoint || ''}" placeholder="/boot">
                </div>
            </div>
            <div class="form-row">
                <div class="form-group">
                    <label>RAID Level</label>
                    <select class="raid-level">
                        <option value="0" ${raid.level === '0' ? 'selected' : ''}>RAID 0 (Stripe)</option>
                        <option value="1" ${raid.level === '1' ? 'selected' : ''}>RAID 1 (Mirror)</option>
                        <option value="5" ${raid.level === '5' ? 'selected' : ''}>RAID 5</option>
                        <option value="6" ${raid.level === '6' ? 'selected' : ''}>RAID 6</option>
                        <option value="10" ${raid.level === '10' ? 'selected' : ''}>RAID 10</option>
                    </select>
                </div>
                <div class="form-group">
                    <label>Device Name</label>
                    <input type="text" class="raid-device" value="${raid.device || 'md0'}" placeholder="md0">
                </div>
            </div>
            <div class="form-row">
                <div class="form-group">
                    <label>File System</label>
                    <select class="raid-fstype">
                        <option value="xfs" ${raid.fstype === 'xfs' ? 'selected' : ''}>XFS</option>
                        <option value="ext4" ${raid.fstype === 'ext4' ? 'selected' : ''}>Ext4</option>
                        <option value="btrfs" ${raid.fstype === 'btrfs' ? 'selected' : ''}>Btrfs</option>
                    </select>
                </div>
                <div class="form-group">
                    <label>Spares</label>
                    <input type="number" class="raid-spares" value="${raid.spares || 0}" min="0">
                </div>
            </div>
            <div class="form-row">
                <div class="form-group" style="width: 100%">
                    <label>Devices (space separated)</label>
                    <input type="text" class="raid-devices" value="${raid.devices?.join(' ') || 'sda1 sdb1'}" placeholder="sda1 sdb1">
                </div>
            </div>
            <div class="form-row storage-toggle-row">
                <div class="storage-toggle-item">
                    <label class="switch">
                        <input type="checkbox" class="raid-encrypted" ${raid.encrypted ? 'checked' : ''}>
                        <span class="slider"></span>
                    </label>
                    <span class="switch-label">Encrypt</span>
                </div>
            </div>
        `;
    }

    getBtrfsFieldsHTML(btrfs = {}) {
        return `
            <div class="form-row">
                <div class="form-group">
                    <label>Disk Type</label>
                    <select class="storage-card-type-select">
                        <option value="partition">Partition</option>
                        <option value="volgroup">Volume Groups</option>
                        <option value="logvol">Logical Volumes</option>
                        <option value="raid">RAID</option>
                        <option value="btrfs" selected>Btrfs</option>
                    </select>
                </div>
                <div class="form-group">
                    <label class="no-required">Mount Point</label>
                    <input type="text" class="btrfs-mount" value="${btrfs.subvol || ''}" placeholder="/ or none">
                </div>
            </div>
            <div class="form-row">
                <div class="form-group">
                    <label>Subvol Name</label>
                    <input type="text" class="btrfs-name" value="${btrfs.name || ''}" placeholder="root">
                </div>
                <div class="form-group" style="width: 100%">
                    <label>Devices (space separated)</label>
                    <input type="text" class="btrfs-devices" value="${btrfs.devices?.join(' ') || 'sda3'}" placeholder="sda3">
                </div>
            </div>
            <div class="form-row">
                <div class="form-group">
                    <label>Data Level</label>
                    <select class="btrfs-level">
                        <option value="">Single</option>
                        <option value="raid0" ${btrfs.level === 'raid0' ? 'selected' : ''}>RAID 0</option>
                        <option value="raid1" ${btrfs.level === 'raid1' ? 'selected' : ''}>RAID 1</option>
                        <option value="raid10" ${btrfs.level === 'raid10' ? 'selected' : ''}>RAID 10</option>
                        <option value="dup" ${btrfs.level === 'dup' ? 'selected' : ''}>Duplicate</option>
                    </select>
                </div>
                <div class="form-group">
                    <label>Metadata Level</label>
                    <select class="btrfs-metalevel">
                        <option value="">Single</option>
                        <option value="raid0" ${btrfs.metaLevel === 'raid0' ? 'selected' : ''}>RAID 0</option>
                        <option value="raid1" ${btrfs.metaLevel === 'raid1' ? 'selected' : ''}>RAID 1</option>
                        <option value="raid10" ${btrfs.metaLevel === 'raid10' ? 'selected' : ''}>RAID 10</option>
                        <option value="dup" ${btrfs.metaLevel === 'dup' ? 'selected' : ''}>Duplicate</option>
                    </select>
                </div>
                <div class="form-group">
                    <label>Label</label>
                    <select class="btrfs-level">
                        <option value="">Single</option>
                        <option value="raid0" ${btrfs.level === 'raid0' ? 'selected' : ''}>RAID 0</option>
                        <option value="raid1" ${btrfs.level === 'raid1' ? 'selected' : ''}>RAID 1</option>
                        <option value="raid10" ${btrfs.level === 'raid10' ? 'selected' : ''}>RAID 10</option>
                        <option value="dup" ${btrfs.level === 'dup' ? 'selected' : ''}>Duplicate</option>
                    </select>
                </div>
                <div class="form-group">
                    <label>Metadata Level</label>
                    <select class="btrfs-metalevel">
                        <option value="">Single</option>
                        <option value="raid0" ${btrfs.metaLevel === 'raid0' ? 'selected' : ''}>RAID 0</option>
                        <option value="raid1" ${btrfs.metaLevel === 'raid1' ? 'selected' : ''}>RAID 1</option>
                        <option value="raid10" ${btrfs.metaLevel === 'raid10' ? 'selected' : ''}>RAID 10</option>
                        <option value="dup" ${btrfs.metaLevel === 'dup' ? 'selected' : ''}>Duplicate</option>
                    </select>
                </div>
                <div class="form-group">
                    <label>Label</label>
                    <input type="text" class="btrfs-label" value="${btrfs.label || ''}" placeholder="root">
                </div>
            </div>
        `;
    }

    getFieldsHTMLByType(type, data = {}) {
        switch (type) {
            case 'partition':
                return this.getPartitionFieldsHTML(data);
            case 'volgroup':
                return this.getVolGroupFieldsHTML(data);
            case 'logvol':
                return this.getLogVolFieldsHTML(data);
            case 'raid':
                return this.getRaidFieldsHTML(data);
            case 'btrfs':
                return this.getBtrfsFieldsHTML(data);
            default:
                return '';
        }
    }

    getStorageCardHTML(type, data = {}) {
        const normalizedData = this.normalizeDataForType(type, data);

        return `
            <div class="storage-header" onclick="this.nextElementSibling.classList.toggle('collapsed')">
                <div class="storage-header-main">
                    <span class="storage-type">${this.getStorageCardTitle(type)}</span>
                </div>
                <div class="storage-header-actions">
                    <button type="button" class="remove-btn" onclick="event.stopPropagation(); window.StorageManager.removeStorageCard(this)">Remove</button>
                </div>
            </div>
            <div class="storage-body">
                <div class="storage-dynamic-fields">
                    ${this.getFieldsHTMLByType(type, normalizedData)}
                </div>
            </div>
        `;
    }

    createStorageCard(type = 'partition', data = {}) {
        const container = document.createElement('div');
        container.className = 'storage-config-card';
        container.dataset.storageType = type;
        container.dataset.storagePayload = JSON.stringify(this.normalizeDataForType(type, data));
        container.innerHTML = this.getStorageCardHTML(type, data);
        this.bindStorageCardEvents(container);
        return container;
    }

    addStorageCard(type = 'partition', data = {}) {
        if (!this.storageConfigs) return null;
        const container = this.createStorageCard(type, data);
        this.storageConfigs.appendChild(container);
        return container;
    }

    // Aliases for backward compatibility with existing code paths
    addPartitionRow(partition = {}) {
        return this.addStorageCard('partition', partition);
    }

    addVolGroupRow(volGroup = {}) {
        return this.addStorageCard('volgroup', volGroup);
    }

    addLogVolRow(logVol = {}) {
        return this.addStorageCard('logvol', logVol);
    }

    addRaidRow(raid = {}) {
        return this.addStorageCard('raid', raid);
    }

    addBtrfsRow(btrfs = {}) {
        return this.addStorageCard('btrfs', btrfs);
    }

    serializeCardDataByType(card, type) {
        switch (type) {
            case 'partition':
                return {
                    mountpoint: card.querySelector('.partition-mount')?.value || '',
                    fstype: card.querySelector('.partition-fstype')?.value || 'xfs',
                    size: parseInt(card.querySelector('.partition-size')?.value, 10) || 1024,
                    onDisk: card.querySelector('.partition-ondisk')?.value || '',
                    grow: card.querySelector('.partition-grow')?.checked || false,
                    asPrimary: card.querySelector('.partition-primary')?.checked || false,
                    encrypted: card.querySelector('.partition-encrypted')?.checked || false
                };
            case 'volgroup':
                return {
                    name: card.querySelector('.vg-name')?.value || '',
                    physicalVolumes: card.querySelector('.vg-pvs')?.value?.split(' ').filter(Boolean) || [],
                    pesize: card.querySelector('.vg-pesize')?.value || '4096'
                };
            case 'logvol':
                return {
                    vgname: card.querySelector('.lv-vgname')?.value || '',
                    name: card.querySelector('.lv-name')?.value || '',
                    mountpoint: card.querySelector('.lv-mount')?.value || '',
                    size: parseInt(card.querySelector('.lv-size')?.value, 10) || 4096,
                    fstype: card.querySelector('.lv-fstype')?.value || 'xfs',
                    grow: card.querySelector('.lv-grow')?.checked || false,
                    encrypted: card.querySelector('.lv-encrypted')?.checked || false
                };
            case 'raid':
                return {
                    mountpoint: card.querySelector('.raid-mount')?.value || '',
                    level: card.querySelector('.raid-level')?.value || '1',
                    device: card.querySelector('.raid-device')?.value || 'md0',
                    fstype: card.querySelector('.raid-fstype')?.value || 'xfs',
                    devices: card.querySelector('.raid-devices')?.value?.split(' ').filter(Boolean) || [],
                    spares: parseInt(card.querySelector('.raid-spares')?.value, 10) || 0,
                    encrypted: card.querySelector('.raid-encrypted')?.checked || false
                };
            case 'btrfs':
                return {
                    subvol: card.querySelector('.btrfs-mount')?.value || '',
                    name: card.querySelector('.btrfs-name')?.value || '',
                    devices: card.querySelector('.btrfs-devices')?.value?.split(' ').filter(Boolean) || [],
                    level: card.querySelector('.btrfs-level')?.value || '',
                    metaLevel: card.querySelector('.btrfs-metalevel')?.value || '',
                    label: card.querySelector('.btrfs-label')?.value || ''
                };
            default:
                return {};
        }
    }

    getCardData(card) {
        const type = card.dataset.storageType || 'partition';
        return this.serializeCardDataByType(card, type);
    }

    updateCardTitle(card) {
        const type = card.dataset.storageType || 'partition';
        const title = card.querySelector('.storage-type');
        if (title) {
            title.textContent = this.getStorageCardTitle(type);
        }
    }

    switchCardType(card, nextType) {
        const currentType = card.dataset.storageType || 'partition';
        if (currentType === nextType) return;

        const preservedData = this.getCardData(card);
        const nextData = this.normalizeDataForType(nextType, preservedData);
        card.dataset.storageType = nextType;
        card.dataset.storagePayload = JSON.stringify(nextData);

        const body = card.querySelector('.storage-body');
        if (body) {
            body.innerHTML = `
                <div class="storage-dynamic-fields">
                    ${this.getFieldsHTMLByType(nextType, nextData)}
                </div>
            `;
        }

        this.bindStorageCardEvents(card);
        this.updateCardTitle(card);
        this.syncStorageConfig();
    }

    bindStorageCardEvents(container) {
        const typeSelect = container.querySelector('.storage-card-type-select');
        if (typeSelect) {
            typeSelect.addEventListener('change', (e) => {
                this.switchCardType(container, e.target.value);
            });
        }

        const inputs = container.querySelectorAll('input, select');
        inputs.forEach(input => {
            if (input.classList.contains('storage-card-type-select')) return;
            input.addEventListener('input', () => {
                this.syncStorageConfig();
            });
            input.addEventListener('change', () => {
                this.syncStorageConfig();
            });
        });
    }

    removeStorageCard(btn) {
        btn.closest('.storage-config-card')?.remove();
        this.syncStorageConfig();
    }

    collectStorageConfigs() {
        const partitions = [];
        const volGroups = [];
        const logVols = [];
        const raids = [];
        const btrfs = [];

        this.storageConfigs?.querySelectorAll('.storage-config-card').forEach(card => {
            const type = card.dataset.storageType || 'partition';
            const data = this.getCardData(card);

            if (type === 'partition') {
                if (data.mountpoint || data.fstype === 'swap') partitions.push(data);
            } else if (type === 'volgroup') {
                if (data.name) volGroups.push(data);
            } else if (type === 'logvol') {
                if (data.vgname && data.name) logVols.push(data);
            } else if (type === 'raid') {
                if (data.mountpoint && data.devices.length > 0) raids.push(data);
            } else if (type === 'btrfs') {
                if (data.name || data.subvol) btrfs.push(data);
            }
        });

        return { partitions, volGroups, logVols, raids, btrfs };
    }

    syncStorageConfig() {
        if (typeof window.AppState === 'undefined' || !window.AppState.config) return;
        const storageConfig = this.collectStorageConfigs();
        // Sync to the shared AppState.config — this is the single source of truth
        // that getConfigFromForm() reads from. This matches UbuntuCraft's pattern.
        window.AppState.config.storage = window.AppState.config.storage || {};
        window.AppState.config.storage.partitions = storageConfig.partitions;
        window.AppState.config.storage.volGroups = storageConfig.volGroups;
        window.AppState.config.storage.logVols = storageConfig.logVols;
        window.AppState.config.storage.raids = storageConfig.raids;
        window.AppState.config.storage.btrfs = storageConfig.btrfs;
    }

    // Delegating aliases for backward compatibility
    updatePartitionConfig() { this.syncStorageConfig(); }
    updateVolGroupConfig() { this.syncStorageConfig(); }
    updateLogVolConfig() { this.syncStorageConfig(); }
    updateRaidConfig() { this.syncStorageConfig(); }
    updateBtrfsConfig() { this.syncStorageConfig(); }

    loadConfig(storage) {
        if (!storage) return;

        if (storage.autopart) {
            const autoRadio = document.querySelector('input[name="partitionMode"][value="auto"]');
            if (autoRadio) autoRadio.checked = true;
            this.setPartitionMode('auto');
        } else {
            const manualRadio = document.querySelector('input[name="partitionMode"][value="manual"]');
            if (manualRadio) manualRadio.checked = true;
            this.setPartitionMode('manual');
        }

        if (document.getElementById('autopartType') && storage.autopartType) {
            document.getElementById('autopartType').value = storage.autopartType;
        }

        if (this.storageConfigs) {
            this.storageConfigs.innerHTML = '';
        }

        if (storage.partitions && storage.partitions.length > 0) {
            storage.partitions.forEach(part => this.addStorageCard('partition', part));
        }

        if (storage.volGroups && storage.volGroups.length > 0) {
            storage.volGroups.forEach(vg => this.addStorageCard('volgroup', vg));
        }

        if (storage.logVols && storage.logVols.length > 0) {
            storage.logVols.forEach(lv => this.addStorageCard('logvol', lv));
        }

        if (storage.raids && storage.raids.length > 0) {
            storage.raids.forEach(raid => this.addStorageCard('raid', raid));
        }

        if (storage.btrfs && storage.btrfs.length > 0) {
            storage.btrfs.forEach(item => this.addStorageCard('btrfs', item));
        }

        if (
            (!storage.partitions || storage.partitions.length === 0) &&
            (!storage.volGroups || storage.volGroups.length === 0) &&
            (!storage.logVols || storage.logVols.length === 0) &&
            (!storage.raids || storage.raids.length === 0) &&
            (!storage.btrfs || storage.btrfs.length === 0)
        ) {
            this.addDefaultPartitions();
        }

        this.syncStorageConfig();
    }

    addSelectedStorageItem() {
        this.addStorageCard('partition');
        this.syncStorageConfig();
    }

    initUI() {
        const partitionModeRadios = document.querySelectorAll('input[name="partitionMode"]');
        partitionModeRadios.forEach(radio => {
            radio.addEventListener('change', (e) => {
                this.setPartitionMode(e.target.value);
                if (typeof window.configManager !== 'undefined') {
                    window.configManager.config.storage.autopart = (e.target.value === 'auto');
                }
            });
        });

        const autoPartType = document.getElementById('autopartType');
        if (autoPartType) {
            autoPartType.addEventListener('change', () => {
                if (typeof window.configManager !== 'undefined') {
                    window.configManager.config.storage.autopartType = autoPartType.value;
                }
            });
        }

        const addStorageItemBtn = document.getElementById('addStorageItemBtn');
        if (addStorageItemBtn) {
            addStorageItemBtn.addEventListener('click', () => this.addSelectedStorageItem());
        }

        ['zerombr', 'clearpartAll', 'initLabel'].forEach(id => {
            const el = document.getElementById(id);
            if (el) {
                el.addEventListener('change', () => {
                    if (typeof window.configManager !== 'undefined') {
                        window.configManager.config.storage[id] = el.checked;
                    }
                });
            }
        });
    }
}

window.StorageManager = StorageManager;
