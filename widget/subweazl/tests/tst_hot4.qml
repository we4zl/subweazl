import QtQuick
import QtTest
import ".."

TestCase {
  id: test
  name: "Hot4Selector"
  when: windowShown
  visible: true
  width: 340
  height: 200

  Hot4Selector {
    id: selector
    width: 320
    height: implicitHeight
    presets: [{name: "One"}, {name: "Two"}, {name: "Three"}, {name: "Four"}]
  }
  SignalSpy { id: playSpy; target: selector; signalName: "playRequested" }
  SignalSpy { id: moveSpy; target: selector; signalName: "moveRequested" }

  function init() {
    playSpy.clear(); moveSpy.clear()
    selector.enabled = true
    selector.presets = [{name: "One"}, {name: "Two"}, {name: "Three"}, {name: "Four"}]
  }
  function test_click() {
    mouseClick(selector, 60, 18)
    compare(playSpy.count, 1)
    compare(playSpy.signalArguments[0][0], 1)
    compare(moveSpy.count, 0)
  }
  function dragToFourth() {
    mousePress(selector, 70, 18)
    for (var i = 1; i <= 12; i++) mouseMove(selector, 70 + i * 14, 18 + i * 3.5, 20)
    mouseRelease(selector, 238, 60)
  }
  function test_drag() {
    dragToFourth()
    compare(moveSpy.count, 1)
    compare(moveSpy.signalArguments[0][0], 1)
    compare(moveSpy.signalArguments[0][1], 4)
    compare(playSpy.count, 0)
    var tile = findChild(selector, "hot4Tile1")
    compare(tile.x, 0); compare(tile.y, 0)
  }
  function test_emptyDestination() {
    selector.presets = [{name: "One"}, {name: "Two"}, {name: "Three"}, {name: ""}]
    dragToFourth()
    compare(moveSpy.count, 1)
    compare(moveSpy.signalArguments[0][1], 4)
  }
  function test_disabled() {
    selector.enabled = false
    mouseClick(selector, 60, 18)
    compare(playSpy.count, 0)
  }
}
