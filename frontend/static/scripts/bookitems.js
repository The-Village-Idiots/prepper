/*
 * bookitems.js -- extra items form handling
 * Copyright (C) Ethan Marshall 2023
 * Part of A-Level Computing 2024
 */

let count = 1;

/*
 * add_item opens up the add an extra item modal, which then handles the
 * interaction further.
 */
function add_item(e)
{
	e.preventDefault();

	$("#add_item_modal").modal("show");
}

/*
 * cancel_add_item dismisses the add item modal.
 */
function cancel_add_item()
{
	$("#add_item_modal").modal("hide");
}

/*
 * do_add_item performs the addition of the item to the items form, changing
 * all required attributes.
 */
function do_add_item(id, name, quantity)
{
	/* do cloning */
	var tmpl = $("#extra_items_template").clone();
	$("#extra_items_body").append(tmpl.html());

	/* fixup attributes */
	$("#new_item_id").text(count).attr("id", "extra_" + id + "_id");
	$("#new_item_name").text(name).attr("id", "extra_" + id + "_name");
	$("#new_item_quantity")
		.attr("id", "extra_" + id + "_name")
		.attr("name", "eqty_" + id)
		.attr("max", quantity);

	/* close dialog */
	count++;
	cancel_add_item();
}
