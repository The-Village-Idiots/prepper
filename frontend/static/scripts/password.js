/*
 * password.js -- Password reset validation handling
 * Copyright (C) Ethan Marshall 2023
 * Part of A-Level Computing 2024
 */

function update_form() {
	var a = $("#new_password").val();
	var b = $("#repeat_password").val();

	console.log(a + " | " + b);

	if (a == b && a != "" && b != "") {
		$("#submitBtn").prop("disabled", false);
		$("#matchMessage").addClass("d-none");
	} else {
		$("#submitBtn").prop("disabled", true);
		$("#matchMessage").removeClass("d-none");
	}
}

function handle_submbit(e) {
	var a = $("#new_password").val();
	var b = $("#repeat_password").val();

	if (a != b || a == "" || b == "")
		e.prevent_default();
}
