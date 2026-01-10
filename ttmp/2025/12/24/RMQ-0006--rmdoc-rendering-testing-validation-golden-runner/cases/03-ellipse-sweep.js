// RMDoc-DSL JS generator: ellipse sweep across multiple Y positions.
//
// Output: a multi-page DSL doc where each page contains:
// - one ellipse centered at (0, y)
// - one rotated square near the lower-right (constant reference)
// - a "page marker" made of N short red dashes near the top-left, where N = page index (1..N)
// - a green bottom marker line (sanity)
//
// This is designed for the device-vs-export debug workflow:
// - upload the rendered PDF to the tablet
// - flip pages and verify that the ellipse moves as expected (top→bottom)
// - use the dash count to identify pages unambiguously
//
// Run locally:
//   go run ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/19-rmdsl-render-to-pdf/main.go \
//     --in  ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/cases/03-ellipse-sweep.js \
//     --out /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/rendering/rmq-0006-ellipse/ellipse-sweep.pdf

function main(params) {
  params = params || {};

  // Default sweep (coarse, covers top->bottom).
  const ys = params.ys || [200, 600, 1000, 1400, 1700];

  const d = rm.doc("ellipse-sweep").notebook().v6();

  ys.forEach((y, i) => {
    const n = i + 1; // dash count for page marker
    const pid = `y${y}`;

    d.page(pid).canvas(rm.space.rm_screen_v6, 1404, 1872)
      .layer("fixture")
        // Ellipse at target Y
        .ellipse({ x: 0, y: y }, 240, 140).stroke(rm.tool.fineliner_2, rm.color.black, 1)

        // Stable rotated square reference
        .rect({ x: 280, y: 1050, w: 320, h: 320 }).rotateDeg(15).stroke(rm.tool.fineliner_2, rm.color.black, 1)

        // Page marker: N short red dashes near top-left (each dash is its own stroke)
        // Dash area: x ≈ -650 .. -450, y ≈ 80, spaced 25px vertically.
      ;

    for (let k = 0; k < n; k++) {
      const yy = 80 + k * 25;
      d.stroke(rm.tool.fineliner_2, rm.color.red, 1).polyline([{ x: -650, y: yy }, { x: -500, y: yy }]);
    }

    // Bottom marker: green line close to bottom
    d.stroke(rm.tool.fineliner_2, rm.color.green, 1).polyline([{ x: -650, y: 1820 }, { x: -450, y: 1820 }]);
  });

  return d.done();
}


