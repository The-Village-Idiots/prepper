/*
 * bookitems.js -- extra items form handling
 * Copyright (C) Ethan Marshall 2023
 * Part of A-Level Computing 2024
 */

const clashes_endpoint = "/api/clashes";

/*
 * submitbtn is used to finally submit the form at the end of the popover
 * lifecycle.
 */
let submitbtn = null;

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

/*
 * show_clashes shows the clashes modal, using the data returned by the API
 * (expected as parsed JSON). This should be treated as the end of control by
 * your function.
 */
function show_clashes(clashes)
{
	if (clashes.length == 0) {
		end_clashes();
		return;
	}

	/* clear out existing clashes */
	$("#clashesBody").empty();

	clashes.forEach((c) => {
		let r = $("#clashesBody").append("<tr></tr>");

		r.append('<td>'+c.equipment_name+'</td>');
		r.append('<td><a href="/book/booking/'+c.booking_id+'">'+c.booking_user+'</a></td>')
		r.append('<td><a class="text-truncate" href="/book/booking/'+c.booking_id+'">'+c.booking_activity+'</a></td>')
		r.append('<td><a href="/book/booking/'+c.booking_id+'">'+c.booking_starts+' - '+c.booking_ends+'</a></td>')
		r.append('<td>'+c.you_quantity+'</td>');
		r.append('<td>'+c.clash_quantity+'</td>');
		r.append('<td>'+c.total_quantity+'</td>');
		r.append('<td class="text-danger">'+c.net_quantity+'</td>');
	});

	$("#clashesModal").modal("show");
}

/*
 * end_clashes ends the clash menu and submits the booking
 */
function end_clashes()
{
	$("#clashesModal").modal('hide');

	$(submitbtn).off('click');
	$(submitbtn).closest("form").submit();
}

/*
 * format_items returns the URI encoded items data with base as the base URL to append to
 */
function format_items(base)
{
	let uri = base;
	items.forEach((item) => {
		uri += "&qty_"+item.ItemID+"="+item.Quantity;
	});

	return uri;
}

function validate_manual(ev)
{
	ev.preventDefault();
	var val = $("#manual-form")[0].reportValidity();
	if (!val) {
		return;
	}

	submitbtn = ev.target;

	let date = $("#date-input").val();
	let stime = $("#stime-input")[0].value;
	let etime = $("#etime-input")[0].value;

	let uri = format_items(clashes_endpoint + "?date="+encodeURIComponent(date) + "&start_time="+encodeURIComponent(stime) + "&end_time="+encodeURIComponent(etime) + "&manual=true");

	var req = new XMLHttpRequest();
	req.open("GET", uri, true);
	req.onreadystatechange = function() {
		if (this.readyState == 4) {
			if (this.status == 200) {
				let dat = JSON.parse(this.responseText);

				show_clashes(dat);
			}
		}
	}
	req.send();
}

function validate_timetable(ev, day, start, end)
{
	ev.preventDefault(true);
	submitbtn = ev.target;

	let weekinput = $(ev.target).closest(".week_commencing_input");
	let week = weekinput[0].value;

	/* note lack of manual query parameter */
	let uri = format_items(clashes_endpoint + "?week_commencing="+week + "&day="+day + "&start_time="+start + "&end_time="+end);
	var req = new XMLHttpRequest();
	req.open("GET", uri, true);
	req.onreadystatechange = function() {
		if (this.readyState == 4) {
			if (this.status == 200) {
				let dat = JSON.parse(this.responseText);

				show_clashes(dat);
			}
		}
	}
	req.send();
}

function testmodal()
{
	show_clashes([
		{
			"equipment_name": "30cm ruler",
			"net_quantity": -3,
			"you_quantity": 3,
			"clash_quantity": 5,
			"booking_id": 1,
			"booking_user": "cbaker",
			"booking_activity": "Year 7 Intro Practical",
			"booking_starts": "09:15:00",
			"booking_ends": "10:00:00",
		},
	]);
}
