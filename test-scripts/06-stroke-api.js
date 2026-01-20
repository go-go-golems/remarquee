// Test 6: Low-level Stroke API
console.log("Test 6: Using the low-level Stroke API");

var doc = new RMDoc();
doc.setTitle("Test 6: Stroke API");

var page = doc.addPage();

// Create a stroke manually
var stroke = new Stroke();
stroke.setTool(2); // Pen
stroke.setColor(0); // Black
stroke.setThickness(2.5);

// Add points to create a wavy line
for (var i = 0; i <= 20; i++) {
    var x = 100 + i * 50;
    var y = 500 + Math.sin(i * 0.5) * 200;
    stroke.addPoint({
        x: x,
        y: y,
        pressure: 100 + Math.sin(i * 0.3) * 50,
        width: 2.5
    });
}

page.addStroke(stroke);

doc.save("test-06-stroke-api.rmdoc");
console.log("Saved to test-06-stroke-api.rmdoc");
