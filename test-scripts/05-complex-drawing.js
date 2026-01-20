// Test 5: Complex drawing - a simple house
console.log("Test 5: Creating a complex drawing");

var doc = new RMDoc();
doc.setTitle("Test 5: Complex Drawing");

var page = doc.addPage();
var canvas = page.getCanvas();

// Draw a house
canvas.setPen({ tool: "pen", color: "black", thickness: 3 });

// House base (rectangle)
canvas.drawRect(400, 600, 600, 400);

// Roof (triangle)
canvas.beginPath();
canvas.moveTo(400, 600);
canvas.lineTo(700, 400);
canvas.lineTo(1000, 600);
canvas.closePath();
canvas.stroke();

// Door
canvas.drawRect(600, 800, 100, 200);

// Windows
canvas.drawRect(500, 700, 80, 80);
canvas.drawRect(900, 700, 80, 80);

// Sun
canvas.setPen({ tool: "pen", color: "yellow", thickness: 2 });
canvas.drawCircle(1200, 300, 80);

// Ground line
canvas.setPen({ tool: "pen", color: "green", thickness: 4 });
canvas.drawLine(100, 1000, 1300, 1000);

doc.save("test-05-complex-drawing.rmdoc");
console.log("Saved to test-05-complex-drawing.rmdoc");
