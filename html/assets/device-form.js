(() => {
    const form = document.getElementById('save-device-form');
    const icon = form.elements.icon;
    const preview = document.getElementById('device-icon-preview');
    const choices = [...form.querySelectorAll('[data-icon-value]')];
    function updatePreview() {
        const choice = choices.find(button => button.dataset.iconValue === icon.value);
        preview.replaceChildren();
        if (choice?.dataset.iconUrl) {
            const image = document.createElement('img');
            image.src = choice.dataset.iconUrl;
            image.alt = '';
            preview.append(image);
        } else {
            preview.textContent = icon.value;
        }
        choices.forEach(button => button.setAttribute('aria-pressed', String(button.dataset.iconValue === icon.value)));
    }
    choices.forEach(button => button.addEventListener('click', () => {
        icon.value = button.dataset.iconValue;
        updatePreview();
        form.querySelector('.icon-picker').open = false;
    }));
    icon.addEventListener('input', updatePreview);
    form.addEventListener('reset', () => setTimeout(updatePreview, 0));
    form.addEventListener('device-loaded', updatePreview);
    const minimum = form.elements.min;
    const maximum = form.elements.max;
    function updateBounds() {
        const visible = ['light', 'zlight'].includes(form.elements.type.value);
        form.querySelector('.device-bounds').hidden = !visible;
        document.getElementById('device-bounds-help').hidden = !visible;
        minimum.disabled = maximum.disabled = !visible;
        validateBounds();
    }
    function validateBounds() {
        maximum.setCustomValidity(!maximum.disabled && minimum.value !== '' && maximum.value !== '' && Number(minimum.value) > Number(maximum.value)
            ? 'Maximum must be greater than or equal to minimum.' : '');
    }
    minimum.addEventListener('input', validateBounds);
    maximum.addEventListener('input', validateBounds);
    form.elements.type.addEventListener('change', updateBounds);
    form.addEventListener('device-loaded', updateBounds);
    form.addEventListener('reset', () => setTimeout(updateBounds, 0));
    form.addEventListener('reset', () => maximum.setCustomValidity(''));
    updateBounds();
})();
