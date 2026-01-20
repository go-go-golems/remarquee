// Test 2: Multiple shapes
console.log("Test 2: Creating multiple shapes");

var doc = new RMDoc();
doc.setTitle("Test 2: Shapes");

var page = doc.addPage();
var canvas = page.getCanvas();

// Draw a rectangle
canvas.drawRect(100, 100, 200, 150);

// Draw a circle
canvas.drawCircle(700, 300, 100);

// Draw a triangle using path
canvas.beginPath();
canvas.moveTo(400, 600);
canvas.lineTo(500, 800);
canvas.lineTo(300, 800);
canvas.closePath();
canvas.stroke();

doc.save("test-02-shapes.rmdoc");
console.log("Saved to test-02-shapes.rmdoc");
