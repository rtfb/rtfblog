QUnit.module("Basic Tests");

function appendTestInputElem(id, name, value) {
    var fixture = document.getElementById("qunit-fixture");
    input = document.createElement("input");
    input.id = id;
    input.name = name;
    input.value = value;
    fixture.appendChild(input);
}

QUnit.test("inputToUri", assert => {
    appendTestInputElem('id', 'website', 'http');
    assert.equal(inputToUri("website"), "website=http");
});

QUnit.module("form validation", {
    beforeEach: function(assert) {
        this.alertMsg = null;
        this.origAlert = myAlert;
        myAlert = function(msg) {
            this.alertMsg = msg;
        }
    },
    afterEach: function() {
        myAlert = this.origAlert;
        this.alertMsg = null;
        this.origAlert = null;
    }
});

QUnit.test("completed comment form", assert => {
    appendTestInputElem('name', '', 'name');
    appendTestInputElem('email', '', 'email');
    assert.equal(validateCommentForm(), true);
});

QUnit.test("incomplete comment form", assert => {
    appendTestInputElem('name', '', '');
    assert.equal(validateCommentForm(), false);
    assert.equal(this.alertMsg, "Name field is mandatory.");
    elt('name').value = 'xxx';
    appendTestInputElem('email', '', '');
    assert.equal(validateCommentForm(), false);
    assert.equal(this.alertMsg, "Email field is mandatory.");
});
