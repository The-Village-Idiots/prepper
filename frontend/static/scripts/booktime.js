/*
 * bookitems.js -- extra items form handling
 * Copyright (C) Ethan Marshall 2023
 * Part of A-Level Computing 2024
 */

/*
 * update_commencing updates all the nested form elements' embedded fields.
 * This is done so that only one week commencing field is required.
 */
function update_commencing()
{
	var v = $("#week_commencing").val();

	$("form").each(function() {
		$(this).children(".week_commencing_input").val(v);
	});
}
