const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

test('dashboard updates devices with punctuation in their names', () => {
    const elements = new Map();
    const irButtons = ['power', 'mute'].map(command => ({
        dataset: { device: 'remote/room:1', command },
        style: {},
    }));
    const document = {
        querySelectorAll(selector) {
            assert.equal(selector, '.ir-command');
            return irButtons;
        },
        getElementById(id) {
            if (!elements.has(id)) elements.set(id, { id, text: '' });
            return elements.get(id);
        },
    };
    let refresh;
    const posts = [];
    function $(element) {
        if (element === document) return { ready(callback) { callback(); } };
        if (typeof element === 'string') {
            assert.equal(element, '#add-rule-button');
            return { click() {} };
        }
        return {
            length: 1,
            text(value) { element.text = value; return this; },
            addClass() { return this; },
            removeClass() { return this; },
        };
    }
    const name = 'cbus/lighting:living.room [lamp]';
    const readings = {
        'remote/room:1': { Type: 'IRControl', last_command: 'power' },
        [name]: { Type: 'light', brightness: 50.5 },
        'CBUS Light': { Type: 'light', brightness: 40 },
        'Wohnzimmer-Licht': { Type: 'light', value: 0 },
        'Legacy group': { Type: 'light', value: 42.6 },
        'Unknown light': { Type: 'light' },
        'Invalid light': { Type: 'light', brightness: 'invalid' },
        'Zero light': { Type: 'light', brightness: 0, value: 70 },
        'sensor/room:1': {
            Type: 'sensor',
            Values: { temperature: { value: 21.4, LastChanged: new Date().toISOString(), History: [] } },
        },
        'plug/room:1': { Type: 'plug', value: 1 },
    };
    $.get = (url, data, callback) => { assert.equal(url, '/api/status'); callback(readings); };
    $.post = (url, data) => posts.push({ url, data: JSON.parse(data) });
    const context = { $, document, setInterval(callback) { refresh = callback; } };
    vm.createContext(context);
    vm.runInContext(fs.readFileSync(__dirname + '/../html/assets/dashboard.js', 'utf8'), context);
    refresh();
    assert.equal(irButtons[0].style.backgroundColor, '#bf616a');
    assert.equal(irButtons[1].style.backgroundColor, '');
    readings['remote/room:1'].last_command = 'mute';
    refresh();
    assert.equal(irButtons[0].style.backgroundColor, '');
    assert.equal(irButtons[1].style.backgroundColor, '#bf616a');
    readings['remote/room:1'].last_command = '';
    refresh();
    assert.equal(irButtons[1].style.backgroundColor, '');
    assert.equal(elements.get('value_' + name).text, '51%');
    assert.equal(elements.get('value_CBUS Light').text, '40%');
    assert.equal(elements.get('value_Wohnzimmer-Licht').text, '0%');
    assert.equal(elements.get('value_Legacy group').text, '43%');
    assert.equal(elements.get('value_Unknown light').text, '--');
    assert.equal(elements.get('value_Invalid light').text, '--');
    assert.equal(elements.get('value_Zero light').text, '0%');
    assert.equal(elements.get('sensor/room:1_temperature').text, 21.4);
    assert.equal(elements.get('value_plug/room:1').text, 'on');
    context.switchDevice(name, '', '+10');
    assert.deepEqual(posts[0], {
        url: '/api/switch', data: { device: name, key: '', value: '+10', duration: 1 },
    });
});
