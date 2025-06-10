$(document).ready(function() {
    setInterval(function () {
        $.get('/api/status', null, function(data, status, jqXHR) {
            const now = new Date()
            for (const name in data) {
                if (data[name].Type === 'plug') {
                    $("#value_" + name).text(data[name].value ? "on" : "off");
                } else if (data[name].Type === 'sensor') {
                    for (let key in data[name].Values) {
                        const prec = (key === "temperature" ? 10 : 1);
                        const lastChange = new Date(data[name].Values[key].LastChanged);
                        const age = (now - lastChange) / 1000 / 60; // age in minutes
                        let value = data[name].Values[key].value;
                        if (data[name].Values[key].Since !== undefined && data[name].Values[key].Since !== null) {
                            let last = new Date(data[name].Values[key].Since);
                            let since = (now - last) / 1000;
                            let sinceStr = Math.round(since) + "s";
                            if (since > 60) {
                                since = since / 60;
                                sinceStr = Math.round(since) + "m";
                                if (since > 60) {
                                    since = since / 60;
                                    sinceStr = Math.round(since) + "h";
                                    if (since > 24) {
                                        since = since / 24;
                                        sinceStr = Math.round(since) + "d";
                                    }
                                }
                            }
                            value = sinceStr
                        }

                        if (value === null) {
                            $(`#${name}_${key}`).text("--")
                        } else if (typeof value === 'number') {
                            $(`#${name}_${key}`).text(Math.round(value * prec) / prec);
                        } else {
                            $(`#${name}_${key}`).text(value);
                        }
                        if (age > 60) {
                            $("#" + name + "_" + key).addClass('outdated');
                        } else {
                            $("#" + name + "_" + key).removeClass('outdated');
                        }

                        if (data[name].Values[key].History != null) {
                            let currentDate = new Date(data[name].Values[key].LastChanged)
                            let previousTime = new Date(currentDate.getTime() - 25 * 60 * 1000);

                            if (key === "temperature") {
                                let trend = 0;
                                for (let i = data[name].Values[key].History.length - 1; i >= 0; i--) {
                                    if (new Date(data[name].Values[key].History[i].Time) <= previousTime) {
                                        trend = data[name].Values[key].History[i].Value - data[name].Values[key].value;
                                        break;
                                    }
                                }

                                if (trend < -0.5) {
                                    $("#trend_" + name + "_" + key).text("🔺");
                                } else if (trend > 0.5) {
                                    $("#trend_" + name + "_" + key).text("🔻");
                                } else {
                                    $("#trend_" + name + "_" + key).text("");
                                }
                            }

                        }
                    }
                } else if (data[name].Type === 'temperature') {
                    /* deprecated */
                    $("#value_" + name).text(Math.round(data[name].value*10)/10);
                    if (data[name].Humidity !== 0) {
                        $("#humidity_" + name).text(Math.round(data[name].Humidity));
                    }
                    // age of value in minutes
                    let age = ((new Date() - (new Date(data[name].lastUpdate)))) / 1000 / 60;
                    if (age > 180) {
                        $("#value_" + name).text("--");
                        $("#value_" + name).addClass('outdated');
                    } else if (age > 60) {
                        $("#value_" + name).addClass('outdated');
                    } else {
                        $("#value_" + name).removeClass('outdated');
                    }

                    if (data[name].history != null) {
                        let currentDate = new Date(data[name].history[data[name].history.length - 1].Time)
                        let previousTime = new Date(currentDate - 30 * 60 * 1000);

                        let trend = 0;
                        for (let i = data[name].history.length - 1; i >= 0; i--) {
                            if (new Date(data[name].history[i].Time) < previousTime) {
                                trend = data[name].history[i].Temperature - data[name].value;
                                break;
                            }
                        }

                        if (trend < -0.5) {
                            $("#trend_" + name).text("🔺");
                        } else if (trend > 0.5) {
                            $("#trend_" + name).text("🔻");
                        } else {
                            $("#trend_" + name).text("");
                        }
                    }
                } else if (data[name].Type === 'door-sensor') {
                    $("#value_" + name).text(data[name].value ? "open" : "closed");
                } else {
                    $("#value_" + name).text(Math.round(data[name].value) + '%');
                }
            }
        }, "json");
    }, 1000);
    $("#add-rule-button").click(function () {
        addRule();
    });
});

function switchDevice(device, value) {
    var data = {
        device: device,
        value: value.toString(),
        duration: 1,
    };
    $.post('/api/switch', JSON.stringify(data), null, 'json');
}

function addRule() {
    $.get('/rules/add-single-use', function (data) {
        $("#popup-content").html(data);
        $("#popup").show();
    }, "html");
}

