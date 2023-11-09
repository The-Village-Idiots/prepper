/*
 * dashboard.js -- periodically reload the dashboard
 * Copyright (C) Ethan Marshall 2023
 * Part of A-Level Computing 2024
 *
 * Dependencies: nada
 */

// One minute in milliseconds
const reload_interval = 60 * 1000;
// Maximum notifications_count before the polling stops
const max_notifications = 5;
// Endpoint on the server to return JSON
const api_endpoint = "/api/dashboard";

// Number of notifications in the notification area.
let notifications_count = 0;

function reload_error(msg)
{
	$("#reload_failure").removeClass("d-none");
	$("#save_error").text(msg);
}

/*
 * dashboard_reload reloads all data from the API
 */
function dashboard_reload()
{
	if (notifications_count >= max_notifications)
		return;

	var req = new XMLHttpRequest();
	req.open("GET", api_endpoint, true);
	req.onreadystatechange = function() {
		if (this.readyState == 4) {
			if (this.status == 200) {
				try {
					let dat = JSON.parse(this.responseText);

					$("#reload_failure").addClass("d-none");
					$("#current_time").text(dat.time);
					dat.notifications.forEach(function(not) {
						var tmpl = $("#toast_template").clone();
						$("#notification_area").append(tmpl.html());

						$("#new-notification .toast-title").text(not.title);
						$("#new-notification .toast-body").text(not.body);
						$("#new-notification .toast-timing").text(not.fmt_time);
						$("#new-notification .btn-close").on("click", ondismissed);

						$("#new-notification").attr("id", "");
						notifications_count++;

						console.log(not);
					})

				} catch (e) {
					reload_error("Malformed response body: " + e);
				}
				return;
			} else {
				reload_error("HTTP Request Failed with code " + this.status);
			}
		}
	}
	req.send();
}

/*
 * Called on the dismissal of a notifications
 */
function ondismissed()
{
	notifications_count--;
}

/*
 * Called on document load (not on notification load!)
 */
function onload()
{
	console.log("dashboad will update every " + reload_interval/1000 + "s");
	setInterval(function() {
		dashboard_reload();
	}, reload_interval);

	dashboard_reload();
}
