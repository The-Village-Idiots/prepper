/*
 * actitems.js -- add items to activity
 * Copyright (C) Ethan Marshall 2023
 * Part of A-Level Computing 2024
 */

/* count variable is set externally in HTML */

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
 * do_add_item is a modified version of the same function for bookitems.js
 * which marks all these items as core (as that is what is stored about
 * activities).
 */
function do_add_item(id, name, quantity)
{
	/* do cloning */
	var tmpl = $("#item_template").clone();
	$("#items_body").append(tmpl.html());

	/* fixup attributes */
	$("#new_item_id").text(count).attr("id", "item_" + id + "_id");
	$("#new_item_name").text(name).attr("id", "item_" + id + "_name");
	$("#new_item_quantity")
		.attr("id", "item_" + id + "_name")
		.attr("name", "qty_" + id)
		.attr("max", quantity);

	/* close dialog */
	count++;
	cancel_add_item();
}

/*
 * delete_item removes an item from the activity items table.
 */
function delete_item(ev)
{
	let _e = $(ev.srcElement);
	let e = _e.closest("tr");
	let t = _e.closest("tbody");

	let echildren = e.children();
	let id = $(echildren[0]).text();

	e.remove();

	t.children().each((_, row) => {
		let rch = $(row).children();
		let cid = $(rch[0]).text();

		if (cid > id) {
			let nid = cid - 1;
			$(rch[0]).text(nid);
		}
	});

	count--;
}
