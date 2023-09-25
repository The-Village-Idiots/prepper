/*
 * form.js -- JQuery form handling
 * Copyright (C) Ethan Marshall 2023
 * Part of A-Level Computing 2024
 */

/*
 * Returns the form at selector s serialized to API-friendly JSON.
 */
function serialize_json(s)
{
	var arr = $(s).serializeArray();
	var json = {};

	$.map(arr, function(n, _){
		json[n['name']] = n['value'];
	});

	return JSON.stringify(json);
}

/*
 * json_form submits a form with the given query selector via a request to the
 * given route.
 *
 * The callbacks onsuccess and onfail are called in either condition.
 */
function json_form(s, method, route, onsuccess, onfail)
{
	var dat = serialize_json(s);
	var req = new XMLHttpRequest();

	req.open(method, route, true);
	req.setRequestHeader("Content-Type", "application/json");
	req.onreadystatechange = function() {
		if (this.readyState == 4) {
			if (this.status == 200) {
				onsuccess(this.responseText);
				return;
			}

			console.log("Data save failure: " + this.responseText);
			onfail(this.responseText);
		}
	}
	req.send(dat);
}