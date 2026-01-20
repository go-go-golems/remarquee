// Test 3: Different pen settings
console.log("Test 3: Testing different pen settings");

var doc = new RMDoc();
doc.setTitle("Test 3: Pen Settings");

var page = doc.addPage();
var canvas = page.getCanvas();

// Draw with different tools and colors
canvas.setPen({ tool: "pen", color: "black", thickness: 2 });
canvas.drawLine(100, 100, 500, 100);

canvas.setPen({ tool: "pencil", color: "gray", thickness: 3 });
canvas.drawLine(100, 200, 500, 200);

canvas.setPen({ tool: "marker", color: "red", thickness: 5 });
canvas.drawLine(100, 300, 500, 300);

canvas.setPen({ tool: "pen", color: "blue", thickness: 1 });
canvas.drawCircle(700, 400, 80);

doc.save("test-03-pen-settings.rmdoc");
console.log("Saved to test-03-pen-settings.rmdoc");
