(() => {
    const devices = window.configuredDevices;
    const modal = document.getElementById('save-device-modal');
    const form = document.getElementById('save-device-form');
    const name = document.getElementById('device-name-input');
    const type = document.getElementById('device-type-input');
    const room = document.getElementById('device-room-input');
    const protocol = document.getElementById('device-protocol-input');
    const submit = document.getElementById('confirm-save-btn');
    const message = document.getElementById('message');
    let active = null;

    // Keep identifiers stable so rules, groups and panels continue to refer to them.
    name.readOnly = true;
    protocol.required = false;
    name.title = 'Device names are identifiers used by rules, groups and panels.';
    form.querySelector('h2').textContent = 'Edit device';
    type.querySelector('[value="generic-device"]').value = 'device';
    type.querySelector('[value="ir-control"]').value = 'ircontrol';
    const originalTypes = type.innerHTML;
    const originalProtocols = protocol.innerHTML;

    function selectValue(select, value, label) {
        if (![...select.options].some(option => option.value === value)) {
            select.add(new Option(label || value, value));
        }
        select.value = value;
    }

    function openDevice(device) {
            active = { device };
            type.innerHTML = originalTypes;
            protocol.innerHTML = originalProtocols;
            name.value = device.Name;
            room.value = device.Room;
            form.elements.label.value = device.Label || '';
            form.elements.hidden.checked = device.Options?.Hidden ?? false;
            form.elements.icon.value = device.Icon || '';
            form.elements.min.value = device.Options?.Min ?? '';
            form.elements.max.value = device.Options?.Max ?? '';
            selectValue(type, device.Type);
            selectValue(protocol, device.Protocol, 'Default');
            form.dispatchEvent(new Event('device-loaded'));
            document.getElementById('save-device-topic').textContent = device.Topic;
            modal.showModal();
            room.focus();
    }
    document.getElementById('cancel-save-btn').addEventListener('click', () => modal.close());
    modal.addEventListener('close', () => { window.location.assign('/devices'); });

    const selectedName = new URLSearchParams(window.location.search).get('name');
    const selectedDevice = devices.find(device => device.Name === selectedName);
    if (selectedDevice) {
        openDevice(selectedDevice);
    } else if (selectedName === null) {
        window.location.replace('/devices');
    } else {
        message.textContent = 'Device not found. Choose a device from the device list.';
        message.className = 'message error';
        message.hidden = false;
    }

    form.addEventListener('submit', async event => {
        event.preventDefault();
        if (!active || submit.disabled) return;
        const edit = active;
        const values = { name: edit.device.Name, type: type.value, room: room.value.trim(), protocol: protocol.value };
        for (const key of ['label', 'icon', 'min', 'max']) values[key] = form.elements[key].value;
        values.hidden = String(form.elements.hidden.checked);
        submit.disabled = true;
        try {
            const response = await fetch('/devices/edit/save', {
                method: 'POST',
                headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                body: new URLSearchParams(values)
            });
            if (!response.ok) throw new Error((await response.text()).trim() || 'Could not save device.');
            const result = await response.json();
            Object.assign(edit.device, { Type: values.type, Room: values.room, Protocol: values.protocol });
            window.location.assign('/devices?saved=1');
        } catch (error) {
            message.textContent = error.message;
            message.className = 'message error';
            modal.querySelector('.save-modal-topic').textContent = error.message;
        } finally {
            submit.disabled = false;
            message.hidden = false;
        }
    });
})();
