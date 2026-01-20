// Test 1: Simple line drawing
console.log("Test 1: Creating a simple line");

var doc = new RMDoc();
doc.setTitle("Test 1: Simple Line");

var page = doc.addPage();
var canvas = page.getCanvas();

// Draw a diagonal line
canvas.drawLine(100, 100, 500, 500);

doc.save("test-01-simple-line.rmdoc");
console.log("Saved to test-01-simple-line.rmdoc");
