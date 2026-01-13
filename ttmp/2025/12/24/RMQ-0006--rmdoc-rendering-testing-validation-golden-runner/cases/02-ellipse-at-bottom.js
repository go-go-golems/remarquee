// RMDoc-DSL JS generator example (goja).
//
// Run:
//   go run ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/scripts/18-rmdsl-render-to-png/main.go \
//     --in  ./ttmp/2025/12/24/RMQ-0006--rmdoc-rendering-testing-validation-golden-runner/cases/02-ellipse-at-bottom.js \
//     --out /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/rendering/rmq-0006-dsl

function main(params) {
  params = params || {};

  return rm.doc("ellipse-at-bottom-js")
    .notebook().v6()
    .page("p1").canvas(rm.space.rm_screen_v6, 1404, 1872)
    .layer("shapes")
      .ellipse({ x: 0, y: 1500 }, 240, 140).stroke(rm.tool.fineliner_2, rm.color.black, 1)
      .rect({ x: 280, y: 1050, w: 320, h: 320 }).rotateDeg(15).stroke(rm.tool.fineliner_2, rm.color.black, 1)
      .stroke(rm.tool.fineliner_2, rm.color.red, 1).polyline([{ x: -650, y: 50 }, { x: -450, y: 50 }])
      .stroke(rm.tool.fineliner_2, rm.color.green, 1).polyline([{ x: -650, y: 1820 }, { x: -450, y: 1820 }])
    .done();
}


