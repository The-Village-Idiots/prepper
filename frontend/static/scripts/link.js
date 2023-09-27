/*
 * link.js -- iSAMS linking form handling
 * Copyright (C) Ethan Marshall 2023
 * Part of A-Level Computing 2024
 */

function update_linking()
{
	$("#accountsTable tr").filter(function() {
		return true;
	}).show();


	if ($("#usernameInput").val() != "") {
		$("#accountsTable tr").filter(function() {
			return $(this).html().indexOf($("#usernameInput").val()) === -1;
		}).hide();
	}
}
