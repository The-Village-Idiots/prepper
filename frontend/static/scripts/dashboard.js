/*
 * dashboard.js -- periodically reload the dashboard
 * Copyright (C) Ethan Marshall 2023
 * Part of A-Level Computing 2024
 *
 * Dependencies: nada
 */

// One minute in milliseconds
const reload_interval = 60 * 1000;

function onload()
{
	console.log("dashboad will reload every " + reload_interval/1000 + "s")
	setTimeout(function() {
		location.reload();
	}, reload_interval);
}
