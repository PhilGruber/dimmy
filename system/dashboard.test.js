const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

test('dashboard updates devices with punctuation in their names', () => {
    const elements = new Map();
    const document = {
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
        [name]: { Type: 'light', value: 50.5 },
        'CBUS Light': { Type: 'light', value: 40 },
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
    assert.equal(elements.get('value_' + name).text, '51%');
    assert.equal(elements.get('value_CBUS Light').text, '40%');
    assert.equal(elements.get('sensor/room:1_temperature').text, 21.4);
    assert.equal(elements.get('value_plug/room:1').text, 'on');
    context.switchDevice(name, '', '+10');
    assert.deepEqual(posts[0], {
        url: '/api/switch', data: { device: name, key: '', value: '+10', duration: 1 },
    });
});
