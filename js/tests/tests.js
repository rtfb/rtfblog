module("Basic Tests");

function appendTestInputElem(name, value) {
    input = document.createElement("input");
    input.name = name;
    input.value = value;
    document.body.appendChild(input);
}

test("inputToUri", function() {
    appendTestInputElem('website', 'http');
    equal(inputToUri("website"), "website=http");
});
