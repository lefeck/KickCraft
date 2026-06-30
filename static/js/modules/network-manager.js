// Network Manager Module

class NetworkManager {
    constructor() {
        this.deviceContainer = null;
        this.deviceIndex = 1; // Start from 1 since 0 is the default template
        this.init();
        this.initUI();
    }

    init() {
        this.deviceContainer = document.getElementById('networkDevices');
        const defaultCard = document.getElementById('defaultNetworkDevice');
        
        if (this.deviceContainer) {
            // Clone the default card and make it visible
            if (defaultCard) {
                const clone = defaultCard.cloneNode(true);
                clone.id = '';
                clone.style.display = 'block';
                
                // Update the index for all input IDs in the cloned card
                this.updateCardIndices(clone, 0);
                
                this.deviceContainer.appendChild(clone);
                
                // Bind events to the cloned card
                this.bindDeviceEvents(clone, 0);
            }
        }
    }
    
    updateCardIndices(card, index) {
        // Update device type label
        const typeLabel = card.querySelector('.device-type');
        if (typeLabel) {
            typeLabel.textContent = `Network Device ${index + 1}`;
        }
        
        // Update all input/select IDs
        card.querySelectorAll('[id]').forEach(el => {
            const oldId = el.id;
            // Replace -0, -1, etc with the new index
            el.id = oldId.replace(/-(\d+)$/, `-${index}`);
        });
        
        // Update for="..." labels
        card.querySelectorAll('[for]').forEach(el => {
            const oldFor = el.getAttribute('for');
            el.setAttribute('for', oldFor.replace(/-(\d+)$/, `-${index}`));
        });
        
        card.dataset.index = index;
    }

    addDevice(config = {}) {
        const defaultCard = document.getElementById('defaultNetworkDevice');
        if (!defaultCard || !this.deviceContainer) return null;
        
        // Clone the template
        const clone = defaultCard.cloneNode(true);
        clone.id = '';
        clone.style.display = 'block';
        
        const index = this.deviceIndex;
        
        // Apply all config values to form fields
        const setValue = (selector, value) => {
            const el = clone.querySelector(selector);
            if (el && value !== undefined && value !== null && value !== '') {
                el.value = value;
            }
        };
        const setChecked = (selector, checked) => {
            const el = clone.querySelector(selector);
            if (el && checked !== undefined && checked !== null) {
                el.checked = !!checked;
            }
        };
        
        // Basic settings
        setValue('.net-device', config.device);
        setValue('.net-bootproto', config.bootProto || 'dhcp');
        
        // Static IP settings
        setValue('.net-ip', config.ip);
        setValue('.net-netmask', config.netmask);
        setValue('.net-gateway', config.gateway);
        setValue('.net-nameserver', config.nameserver);
        setValue('.net-mtu', config.mtu);
        
        // IPv6 settings
        setValue('.net-ipv6', config.ipv6);
        setValue('.net-ipv6gateway', config.ipv6Gateway);
        
        // Advanced settings
        setChecked('.net-onboot', config.onBoot);
        setChecked('.net-activate', config.activate);
        setChecked('.net-nodns', config.noDns);
        setChecked('.net-noipv4', config.noIpv4);
        setChecked('.net-noipv6', config.noIpv6);
        setChecked('.net-nodefroute', config.noDefaultRoute);
        
        // Other settings
        setValue('.net-interfacename', config.interfaceName);
        setValue('.net-ethtool', config.ethtool);
        
        // Update indices
        this.updateCardIndices(clone, index);
        
        // Show/hide static options based on bootProto
        const staticOptions = clone.querySelector('.static-options');
        if (staticOptions) {
            staticOptions.classList.toggle('hidden', config.bootProto !== 'static');
        }
        
        this.deviceContainer.appendChild(clone);
        this.deviceIndex++;
        this.bindDeviceEvents(clone, index);
        return clone;
    }

    bindDeviceEvents(card, index) {
        const deleteBtn = card.querySelector('.device-delete');
        if (deleteBtn) {
        deleteBtn.addEventListener('click', () => {
            card.remove();
            // Rebuild AppState.config.networks from remaining .network-device cards
            if (window.AppState?.config) {
                const cards = this.deviceContainer?.querySelectorAll('.network-device[data-index]') || [];
                window.AppState.config.networks = Array.from(cards).map(c => ({
                    device: c.querySelector('.net-device')?.value || '',
                    bootProto: c.querySelector('.net-bootproto')?.value || 'dhcp',
                    ip: c.querySelector('.net-ip')?.value || '',
                    gateway: c.querySelector('.net-gateway')?.value || '',
                    netmask: c.querySelector('.net-netmask')?.value || '',
                    nameserver: c.querySelector('.net-nameserver')?.value || '',
                    mtu: c.querySelector('.net-mtu')?.value || '',
                    onBoot: c.querySelector('.net-onboot')?.checked || false,
                    activate: c.querySelector('.net-activate')?.checked || false,
                    noDns: c.querySelector('.net-nodns')?.checked || false,
                    noIpv4: c.querySelector('.net-noipv4')?.checked || false,
                    noIpv6: c.querySelector('.net-noipv6')?.checked || false,
                    ipv6: c.querySelector('.net-ipv6')?.value || '',
                    ipv6Gateway: c.querySelector('.net-ipv6gateway')?.value || '',
                    noDefaultRoute: c.querySelector('.net-nodefroute')?.checked || false,
                    interfaceName: c.querySelector('.net-interfacename')?.value || '',
                    ethtool: c.querySelector('.net-ethtool')?.value || '',
                }));
            }
        });
        }
        const bootProto = card.querySelector('.net-bootproto');
        const staticOptions = card.querySelector('.static-options');
        if (bootProto && staticOptions) {
            bootProto.addEventListener('change', () => {
                staticOptions.classList.toggle('hidden', bootProto.value !== 'static');
            });
        }
        card.querySelectorAll('input, select').forEach(input => {
            input.addEventListener('change', () => this.updateDevice(index));
            input.addEventListener('input', () => this.updateDevice(index));
        });
    }

    updateDevice(index) {
        if (!window.AppState?.config) return;

        const card = this.deviceContainer?.querySelector(`.network-device[data-index="${index}"]`);
        if (!card) return;
        
        const device = {
            device: card.querySelector('.net-device')?.value || '',
            bootProto: card.querySelector('.net-bootproto')?.value || 'dhcp',
            ip: card.querySelector('.net-ip')?.value || '',
            gateway: card.querySelector('.net-gateway')?.value || '',
            netmask: card.querySelector('.net-netmask')?.value || '',
            nameserver: card.querySelector('.net-nameserver')?.value || '',
            mtu: card.querySelector('.net-mtu')?.value || '',
            onBoot: card.querySelector('.net-onboot')?.checked || false,
            activate: card.querySelector('.net-activate')?.checked || false,
            noDns: card.querySelector('.net-nodns')?.checked || false,
            noIpv4: card.querySelector('.net-noipv4')?.checked || false,
            noIpv6: card.querySelector('.net-noipv6')?.checked || false,
            ipv6: card.querySelector('.net-ipv6')?.value || '',
            ipv6Gateway: card.querySelector('.net-ipv6gateway')?.value || '',
            noDefaultRoute: card.querySelector('.net-nodefroute')?.checked || false,
            interfaceName: card.querySelector('.net-interfacename')?.value || '',
            ethtool: card.querySelector('.net-ethtool')?.value || ''
        };
        
        // Sync to the shared AppState.config — matches UbuntuCraft's pattern.
        window.AppState.config.networks = window.AppState.config.networks || [];
        window.AppState.config.networks[index] = device;
    }

    loadConfig(networks) {
        if (!this.deviceContainer) return;
        
        // Remove all existing cards except the template
        const defaultCard = document.getElementById('defaultNetworkDevice');
        this.deviceContainer.innerHTML = '';
        if (defaultCard) {
            this.deviceContainer.appendChild(defaultCard);
            defaultCard.style.display = 'none';
        }
        
        this.deviceIndex = 0;
        
        if (networks && networks.length > 0) {
            networks.forEach(net => {
                this.addDevice(net);
            });
            // Sync the freshly-loaded devices into AppState.config so that
            // getConfigFromForm() reads the correct state immediately.
            if (window.AppState?.config) {
                window.AppState.config.networks = networks;
            }
        } else {
            // Add one default device
            this.addDevice();
        }
    }

    initUI() {
        // Bind Add Network Device button
        const addNetworkBtn = document.getElementById('addNetworkBtn');
        if (addNetworkBtn) {
            addNetworkBtn.addEventListener('click', () => this.addDevice());
        }
    }
}

window.NetworkManager = NetworkManager;
