/*
 * add-items.js -- inventory bulk add handling
 * Copyright (C) Ethan Marshall 2023
 * Part of A-Level Computing 2024
 */

/*
 * current_id is the current id of the object, incremented *after* the
 * insertion.
 */
let current_id = 1;

/*
 * working_id is a hack to pass arguments to a callback which doesn't accept
 * them. it is the ID of the currently saving form.
 */
let working_id = 0;

/*
 * add_item copies the new row template and adds the necessary attributes to
 * the child elements.
 */
function add_item()
{
	/* copy out template */
	var tmpl = $("#newItemTemplate").clone();
	$("#items-body").append(tmpl.html());

	/* generate and update ID */
	var id = "row-" + current_id;
	var name = "New Item " + (current_id - 1);
	current_id++;

	/* fixup child attributes */
	$("#row-0").attr("id", id);
	$("#newtoggle").attr("data-bs-target", "#" + id).attr("id", id + "toggle");
	$("#newrowid").text(current_id - 1).attr("id", id + "id")
	$("#newrowname").text(name).attr("id", id + "name")
	$("#newrowstatus").attr("id", id + "status")
	$("#newrowquantity").text("1").attr("id", id + "quantity")

	/* create item in database */
	create_item(id, name);
}

// create_item creates an item and fills in required stuff for the API, calling
// success on success or fail on failure with the failure message.
function create_item(formid, name)
{
	var obj = {
		"name": name,
		"description": name,
		"available": true,
		"quantity": 1,
	};
	var dat = JSON.stringify(obj);
	var req = new XMLHttpRequest();

	req.open("POST", "/api/item/create")
	req.setRequestHeader("Content-Type", "application/json");
	req.onreadystatechange = function() {
		if (this.readyState == 4) {
			$("#" + formid + "status .saving-icon").addClass("d-none");
			$("#" + formid + "status .error-icon").addClass("d-none");

			try {
				var response = JSON.parse(this.responseText);
			} catch {
				$("#" + formid + "status .error-icon").removeClass("d-none");
				return;
			}

			if (this.status == 200) {
				$("#" + formid + "status .ok-icon").removeClass("d-none");

				// Set real ID for API use
				$("#" + formid + " .real-id").html(response.ID);
				return;
			}

			// Failure handle
			console.log("Data save failure: " + this.responseText);

			$("#" + formid + " .error-title").text(response.error);
			$("#" + formid + " .error-body").text(response.message);

			$("#" + formid + "status .error-icon").removeClass("d-none");
		}
	}

	req.send(dat);
}

function on_update_success()
{
	$("#" + working_id + "status .saving-icon").addClass("d-none");
	$("#" + working_id + "status .ok-icon").removeClass("d-none");
}

function on_update_fail()
{
	$("#" + working_id + "status .saving-icon").addClass("d-none");
	$("#" + working_id + "status .error-icon").removeClass("d-none");
}

// update_item updates the item for which the form contianing e points to.
function update_item(e)
{
	e.preventDefault();

	var frm = $(e.srcElement);
	var id = real_item_id(frm);
	var formid = item_id(frm);

	// If we don't have a real ID yet, make the item to obtain one.
	if (id == "") {
		create_item(formid, $("#" + formid + " .name-input").val());
	}

	var val = frm[0].checkValidity();
	frm.addClass("was-validated");
	if (!val) {
		$("#" + working_id + "status .ok-icon").addClass("d-none");
		$("#" + working_id + "status .error-icon").removeClass("d-none");
		return;
	}

	// Update status icons
	$("#" + formid + "status .error-icon").addClass("d-none");
	$("#" + formid + "status .ok-icon").addClass("d-none");
	$("#" + formid + "status .saving-icon").removeClass("d-none");

	// Update table fields
	$("#" + formid + "name").text($("#" + formid + " .name-input").val());
	$("#" + formid + "quantity").text($("#" + formid + " .quantity-input").val());

	// Collapse the collapse
	var bsCollapse = new bootstrap.Collapse("#" + formid);
	bsCollapse.hide();

	working_id = formid;
	setTimeout(function() {
		json_form("#" + formid, "POST", "/api/item/"+id+"/edit", on_update_success, on_update_fail);
	}, 100);
}

// item_id returns a valid DOM ID for the row entry based on the event source
// passed. For a valid return value, this must be an event originated from
// within the edit form.
function item_id(e)
{
	var ev = $(e);

	return ev.attr("id");
}

// real_item_id attempts to find the real item ID for the API of this item.
function real_item_id(e)
{
	var ev = $(e);

	return ev.children(".real-id").text();
}
