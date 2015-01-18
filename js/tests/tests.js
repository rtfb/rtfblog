module("Basic Tests");

function appendTestInputElem(id, name, value) {
    var fixture = document.getElementById("qunit-fixture");
    input = document.createElement("input");
    input.id = id;
    input.name = name;
    input.value = value;
    fixture.appendChild(input);
}

test("inputToUri", function() {
    appendTestInputElem('id', 'website', 'http');
    equal(inputToUri("website"), "website=http");
});

module("form validation", {
    setup: function() {
        this.alertMsg = null;
        this.origAlert = myAlert;
        var that = this;
        myAlert = function(msg) {
            that.alertMsg = msg;
        }
    },
    teardown: function() {
        myAlert = this.origAlert;
        this.alertMsg = null;
        this.origAlert = null;
    }
});

test("completed comment form", function() {
    appendTestInputElem('name', '', 'name');
    appendTestInputElem('email', '', 'email');
    equal(validateCommentForm(), true);
});

test("incomplete comment form", function() {
    appendTestInputElem('name', '', '');
    equal(validateCommentForm(), false);
    equal(this.alertMsg, "Name field is mandatory.");
    elt('name').value = 'xxx';
    appendTestInputElem('email', '', '');
    equal(validateCommentForm(), false);
    equal(this.alertMsg, "Email field is mandatory.");
});
