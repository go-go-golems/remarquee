// Test 4: Multiple pages
console.log("Test 4: Creating document with multiple pages");

var doc = new RMDoc();
doc.setTitle("Test 4: Multiple Pages");

// Page 1: Simple line
var page1 = doc.addPage();
page1.setTemplate("Blank");
var canvas1 = page1.getCanvas();
canvas1.drawLine(100, 100, 1300, 100);

// Page 2: Circle
var page2 = doc.addPage();
page2.setTemplate("Lined");
var canvas2 = page2.getCanvas();
canvas2.drawCircle(700, 900, 200);

// Page 3: Rectangle
var page3 = doc.addPage();
page3.setTemplate("Grid");
var canvas3 = page3.getCanvas();
canvas3.drawRect(300, 300, 800, 600);

console.log("Created " + doc.getPageCount() + " pages");

doc.save("test-04-multiple-pages.rmdoc");
console.log("Saved to test-04-multiple-pages.rmdoc");
