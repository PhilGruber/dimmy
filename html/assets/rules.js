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
        return `<option value="">Choose a device</option>${items.map(device =>
            `<option value="${escapeHtml(device.name)}"${device.name === selected ? " selected" : ""}>${escapeHtml(`${device.icon} ${device.label}`.trim())}</option>`
        ).join("")}`;
    };

    const escapeHtml = value => String(value).replace(/[&<>'"]/g, character => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;" }[character]));
    const deviceFor = name => devices.find(device => device.name === name);

    function setFields(row, kind, selected = "") {
        const device = deviceFor(row.querySelector(".device").value);
        const fields = fieldsFor(device, kind);
        row.querySelector(".field").innerHTML = `<option value="">Choose a field</option>${fields.map(field => `<option value="${escapeHtml(field)}"${field === selected ? " selected" : ""}>${escapeHtml(field)}</option>`).join("")}`;
    }

    function addTrigger(trigger = {}) {
        const row = document.createElement("div");
        row.className = "rule-row trigger-row";
        row.innerHTML = `<select class="device" aria-label="Sensor device">${optionsFor("triggers", trigger.device)}</select><select class="field" aria-label="Sensor field"></select><select class="operator" aria-label="Condition"><option value="==">is</option><option value="!=">is not</option><option value=">">is greater than</option><option value=">=">is at least</option><option value="<">is less than</option><option value="<=">is at most</option></select><input class="condition-value" aria-label="Condition value" placeholder="Value"><button type="button" class="remove-row" aria-label="Remove sensor">&times;</button>`;
        triggerRows.append(row);
        setFields(row, "triggers", trigger.key);
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
        setFields(row, "receivers", receiver.key);
        row.querySelector(".receiver-value").value = receiver.value ?? "";
        row.querySelector(".device").addEventListener("change", () => setFields(row, "receivers"));
        row.querySelector(".remove-row").addEventListener("click", () => row.remove());
    }

    function openRule(index = null) {
        editingIndex = index;
        const rule = index === null ? { triggers: [{}], receivers: [{}] } : rules[index];
        title.textContent = index === null ? "Add rule" : "Edit rule";
        triggerRows.replaceChildren(); receiverRows.replaceChildren();
        (rule.triggers || []).forEach(addTrigger); (rule.receivers || []).forEach(addReceiver);
        formError.hidden = true;
        modal.hidden = false; modal.setAttribute("aria-hidden", "false");
        triggerRows.querySelector("select")?.focus();
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
        const updatedRules = [...rules];
        const rule = { triggers, receivers };
        if (editingIndex === null) updatedRules.push(rule); else updatedRules[editingIndex] = rule;
        try {
            await saveRules(updatedRules);
            window.location.reload();
        } catch (error) { showError(error.message || "Could not save the rule."); }
    });
})();
