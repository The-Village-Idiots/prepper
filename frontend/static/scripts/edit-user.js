/*
 * edit-user.js -- edit user form handling
 * Copyright (C) Ethan Marshall 2023
 * Part of A-Level Computing 2024
 *
 * Dependencies: form.js jquery.min.js
 */

/* Global constant "userid" is defined elsewhere */

function onsuccess()
{
	$("#savingSpinner").addClass("d-none");

	$("#saveFailure").addClass("d-none");
	$("#saveSuccess").removeClass("d-none");

	$("#saveBtn").removeClass("btn-danger");
	$("#saveBtn").addClass("btn-success");

	setTimeout(function() {
		$("#saveBtn").removeClass("btn-success");
		$("#saveSuccess").addClass("d-none");
	}, 2000)
}

function onfail()
{
	$("#savingSpinner").addClass("d-none");

	$("#saveFailure").removeClass("d-none");
	$("#saveSuccess").addClass("d-none");

	$("#saveBtn").removeClass("btn-success");
	$("#saveBtn").addClass("btn-danger");
}

function saveUser(e)
{
	e.preventDefault();

	var val = $("form")[0].checkValidity();
	$("form").addClass("was-validated");
	if (!val)
		return;

	$("#savingSpinner").removeClass("d-none");

	setTimeout(function() {
		json_form("form", "POST", "/api/user/edit/" + userid, onsuccess, onfail);
	}, 100);

	$("form").removeClass("was-validated");
}
