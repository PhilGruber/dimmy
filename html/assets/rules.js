(() => {
    const rules = Array.isArray(window.ruleData) ? window.ruleData : [];
    const devices = Array.isArray(window.ruleDevices) ? window.ruleDevices : [];
    const modal = document.getElementById("rule-modal");
    const form = document.getElementById("rule-form");
    const triggerRows = document.getElementById("trigger-rows");
    const receiverRows = document.getElementById("receiver-rows");
    const title = document.getElementById("rule-modal-title");
    const formError = document.getElementById("form-error");
    const message = document.getElementById("message");
    let editingIndex = null;

    const fieldsFor = (device, kind) => Array.isArray(device?.[kind]) ? device[kind] : [];

    const optionsFor = (kind, selected = "") => {
        const items = devices.filter(device => fieldsFor(device, kind).length);
        const unavailable = selected && !items.some(device => device.name === selected)
            ? `<option value="${escapeHtml(selected)}" selected>${escapeHtml(selected)} (unavailable)</option>` : "";
        return `<option value="">Choose a device</option>${unavailable}${items.map(device =>
            `<option value="${escapeHtml(device.name)}"${device.name === selected ? " selected" : ""}>${escapeHtml(`${device.icon_url ? '' : device.icon} ${device.label}`.trim())}</option>`
        ).join("")}`;
    };

    const escapeHtml = value => String(value).replace(/[&<>'"]/g, character => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;" }[character]));
    const deviceFor = name => devices.find(device => device.name === name);

    function deviceLabel(device, fallback) {
        if (!device) return escapeHtml(fallback);
        const icon = device.icon_url
            ? `<img class="icon" src="${escapeHtml(device.icon_url)}" alt="">`
            : escapeHtml(device.icon);
        return `${icon} <span>${escapeHtml(device.label)}</span>`;
    }

    function addDevicePicker(row) {
        const select = row.querySelector(".device");
        select.hidden = true;
        const picker = document.createElement("details");
        picker.className = "device-picker";
        const summary = document.createElement("summary");
        summary.setAttribute("aria-label", select.getAttribute("aria-label"));
        picker.append(summary);
        const menu = document.createElement("div");
        menu.className = "device-options";
        for (const option of select.options) {
            const button = document.createElement("button");
            button.type = "button";
            button.innerHTML = deviceLabel(deviceFor(option.value), option.textContent);
            button.addEventListener("click", () => {
                select.value = option.value;
                select.dispatchEvent(new Event("change"));
                picker.open = false;
                summary.focus();
            });
            menu.append(button);
        }
        picker.append(menu);
        select.after(picker);
        const update = () => {
            summary.innerHTML = deviceLabel(deviceFor(select.value), select.selectedOptions[0]?.textContent || "Choose a device");
        };
        select.addEventListener("change", update);
        picker.addEventListener("keydown", event => {
            if (event.key === "Escape") {
                event.stopPropagation();
                picker.open = false;
                summary.focus();
            }
        });
        update();
    }

    function setFields(row, kind, selected = "", value = "") {
        const device = deviceFor(row.querySelector(".device").value);
        const fields = fieldsFor(device, kind);
        const unavailable = selected && !fields.includes(selected)
            ? `<option value="${escapeHtml(selected)}" selected>${escapeHtml(selected)} (unavailable)</option>` : "";
        row.querySelector(".field").innerHTML = `<option value="">Choose a field</option>${unavailable}${fields.map(field => `<option value="${escapeHtml(field)}"${field === selected ? " selected" : ""}>${escapeHtml(field)}</option>`).join("")}`;
        const previous = row.querySelector(kind === "triggers" ? ".condition-value" : ".receiver-value");
        const isIR = device?.type === "IRControl" || device?.type === "ircontrol";
        const input = document.createElement(isIR ? "select" : "input");
        input.className = previous.className;
        input.setAttribute("aria-label", previous.getAttribute("aria-label"));
        if (isIR) {
            input.add(new Option("Choose a command", ""));
            const commands = Array.isArray(device.commands) ? device.commands : [];
            if (value !== "" && !commands.includes(value)) input.add(new Option(`${value} (unavailable)`, value));
            for (const command of commands) input.add(new Option(command, command));
        } else {
            input.placeholder = "Value";
        }
        input.value = value;
        previous.replaceWith(input);
    }

    function addTrigger(trigger = {}) {
        const row = document.createElement("div");
        row.className = "rule-row trigger-row";
        row.innerHTML = `<select class="device" aria-label="Trigger device">${optionsFor("triggers", trigger.device)}</select><select class="field" aria-label="Trigger field"></select><select class="operator" aria-label="Condition"><option value="==">is</option><option value="!=">is not</option><option value=">">is greater than</option><option value=">=">is at least</option><option value="<">is less than</option><option value="<=">is at most</option></select><input class="condition-value" aria-label="Condition value" placeholder="Value"><button type="button" class="remove-row" aria-label="Remove trigger">&times;</button>`;
        triggerRows.append(row);
        addDevicePicker(row);
        setFields(row, "triggers", trigger.key, trigger.condition?.value ?? "");
        row.querySelector(".operator").value = trigger.condition?.operator || "==";
        row.querySelector(".condition-value").value = trigger.condition?.value ?? "";
        row.querySelector(".device").addEventListener("change", () => setFields(row, "triggers"));
        row.querySelector(".remove-row").addEventListener("click", () => row.remove());
    }

    function addReceiver(receiver = {}) {
        const row = document.createElement("div");
        row.className = "rule-row receiver-row";
        row.innerHTML = `<select class="device" aria-label="Control device">${optionsFor("receivers", receiver.device)}</select><select class="field" aria-label="Control field"></select><input class="receiver-value" aria-label="Control value" placeholder="Value"><button type="button" class="remove-row" aria-label="Remove control">&times;</button>`;
        receiverRows.append(row);
        addDevicePicker(row);
        setFields(row, "receivers", receiver.key, receiver.value ?? "");
        row.querySelector(".receiver-value").value = receiver.value ?? "";
        row.querySelector(".device").addEventListener("change", () => setFields(row, "receivers"));
        row.querySelector(".remove-row").addEventListener("click", () => row.remove());
    }

    function openRule(index = null, cloning = false) {
        editingIndex = cloning ? null : index;
        const rule = index === null ? { triggers: [{}], receivers: [{}] } : rules[index];
        title.textContent = cloning ? "Clone rule" : index === null ? "Add rule" : "Edit rule";
        triggerRows.replaceChildren(); receiverRows.replaceChildren();
        (rule.triggers || []).forEach(addTrigger); (rule.receivers || []).forEach(addReceiver);
        formError.hidden = true;
        modal.hidden = false; modal.setAttribute("aria-hidden", "false");
        triggerRows.querySelector("summary")?.focus();
    }

    function closeModal() { modal.hidden = true; modal.setAttribute("aria-hidden", "true"); }
    function showError(message) { formError.textContent = message; formError.hidden = false; }

    async function saveRules(updatedRules) {
        const response = await fetch("/api/rules", {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(updatedRules)
        });
        if (!response.ok) throw new Error(await response.text());
    }

    document.querySelector("[data-new-rule]").addEventListener("click", () => openRule());
    document.querySelectorAll("[data-edit-rule]").forEach(button => button.addEventListener("click", () => openRule(Number(button.dataset.editRule))));
    document.querySelectorAll("[data-clone-rule]").forEach(button => button.addEventListener("click", () => openRule(Number(button.dataset.cloneRule), true)));
    document.querySelectorAll("[data-delete-rule]").forEach(button => button.addEventListener("click", async () => {
        const index = Number(button.dataset.deleteRule);
        if (!window.confirm("Delete this rule?")) return;
        button.disabled = true;
        try {
            await saveRules(rules.filter((_, ruleIndex) => ruleIndex !== index));
            window.location.reload();
        } catch (error) {
            message.textContent = error.message || "Could not delete the rule.";
            message.hidden = false;
            button.disabled = false;
        }
    }));
    document.querySelectorAll("[data-close-modal]").forEach(button => button.addEventListener("click", closeModal));
    document.querySelector("[data-add-trigger]").addEventListener("click", () => addTrigger());
    document.querySelector("[data-add-receiver]").addEventListener("click", () => addReceiver());
    document.addEventListener("keydown", event => { if (event.key === "Escape" && !modal.hidden) closeModal(); });

    form.addEventListener("submit", async event => {
        event.preventDefault();
        const triggers = [...triggerRows.children].map(row => ({
            device: row.querySelector(".device").value,
            key: row.querySelector(".field").value,
            active: true,
            condition: { operator: row.querySelector(".operator").value, value: row.querySelector(".condition-value").value }
        }));
        const receivers = [...receiverRows.children].map(row => ({
            device: row.querySelector(".device").value,
            key: row.querySelector(".field").value,
            value: row.querySelector(".receiver-value").value
        }));
        if (!triggers.length || !receivers.length || [...triggers, ...receivers].some(item => !item.device || !item.key) || triggers.some(item => item.condition.value === "") || receivers.some(item => item.value === "")) {
            showError("Choose a device and field, and enter a value for every row.");
            return;
        }
        const invalid = triggers.some(item => !fieldsFor(deviceFor(item.device), "triggers").includes(item.key))
            || receivers.some(item => !fieldsFor(deviceFor(item.device), "receivers").includes(item.key));
        if (invalid) {
            showError("Replace or remove unavailable triggers and controls before saving this rule.");
            return;
        }
        const updatedRules = [...rules];
        const rule = { triggers, receivers };
        if (editingIndex === null) updatedRules.push(rule); else updatedRules[editingIndex] = rule;
        try {
            await saveRules(updatedRules);
            window.location.reload();
        } catch (error) { showError(error.message || "Could not save the rule."); }
    });
})();
