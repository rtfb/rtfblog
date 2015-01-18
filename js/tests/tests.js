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
        window.alert = function(msg) {
            //console.log(msg);
        }
    },
    teardown: function() {
        if (window.hasOwnProperty('alert')) {
            delete window.alert;
        }
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
    appendTestInputElem('email', '', '');
    equal(validateCommentForm(), false);
});
