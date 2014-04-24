module("Basic Tests");

test("inputToUri", function() {
    input = document.createElement("input");
    input.name = 'website';
    input.value = 'http';
    document.body.appendChild(input);
    equal(inputToUri("website"), "website=http");
});
