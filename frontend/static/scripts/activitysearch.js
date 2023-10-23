/*
 * activity.js -- activity table search handling
 * Adapted from link.js
 * Copyright (C) Ethan Marshall 2023
 * Part of A-Level Computing 2024
 */

function update_search()
{
	$("#itemsTable tr").filter(function() {
		return true;
	}).show();


	if ($("#itemSearch").val() != "") {
		$("tr.item-searchable").filter(function() {
			return $(this).html().toLowerCase().indexOf($("#itemSearch").val().toLowerCase()) === -1;
		}).hide();
	}
}
