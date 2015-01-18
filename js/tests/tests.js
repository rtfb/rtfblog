module("Basic Tests");

function appendTestInputElem(name, value) {
    var fixture = document.getElementById("qunit-fixture");
    input = document.createElement("input");
    input.name = name;
    input.value = value;
    fixture.appendChild(input);
}

test("inputToUri", function() {
    appendTestInputElem('website', 'http');
    equal(inputToUri("website"), "website=http");
});
