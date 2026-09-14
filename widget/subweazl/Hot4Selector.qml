pragma ComponentBehavior: Bound
import QtQuick
import QtQuick.Controls

Item {
  id: selector
  property var presets: []
  property int draggedSlot: -1
  property color foreground: palette.windowText
  property color accent: palette.highlight
  property string fontFamily: ""
  property int fontSize: 13
  property real spacing: 6
  property real buttonHeight: 36
  signal playRequested(int slot)
  signal moveRequested(int from, int to)
  implicitHeight: buttonHeight * 2 + spacing

  SystemPalette { id: palette }

  Grid {
    id: grid
    anchors.fill: parent
    columns: 2
    spacing: selector.spacing

    Repeater {
      model: 4
      Item {
        id: slot
        required property int index
        readonly property var preset: selector.presets[index] || {name: "", active: false}
        objectName: "hot4Slot" + (index + 1)
        width: (grid.width - grid.spacing) / 2
        height: selector.buttonHeight
        z: mouse.drag.active ? 2 : 0

        DropArea {
          id: dropArea
          anchors.fill: parent
          keys: ["subweazl-hot4"]
          onDropped: function(drop) {
            if (!drop.source || selector.draggedSlot < 0 || selector.draggedSlot === slot.index) return
            selector.moveRequested(selector.draggedSlot + 1, slot.index + 1)
            drop.acceptProposedAction()
          }
        }

        Rectangle {
          id: tile
          objectName: "hot4Tile" + (slot.index + 1)
          width: slot.width
          height: slot.height
          radius: 4
          enabled: slot.preset.name !== ""
          activeFocusOnTab: enabled
          opacity: enabled ? (mouse.drag.active ? 0.7 : 1) : 0.4
          color: slot.preset.active || dropArea.containsDrag
            ? Qt.rgba(selector.accent.r, selector.accent.g, selector.accent.b, 0.22)
            : Qt.rgba(selector.foreground.r, selector.foreground.g, selector.foreground.b, mouse.containsMouse ? 0.14 : 0.04)
          border.width: 1
          border.color: activeFocus || dropArea.containsDrag || slot.preset.active
            ? selector.accent : Qt.rgba(selector.foreground.r, selector.foreground.g, selector.foreground.b, 0.35)
          Drag.active: mouse.drag.active
          Drag.source: tile
          Drag.keys: ["subweazl-hot4"]
          Drag.hotSpot.x: width / 2
          Drag.hotSpot.y: height / 2
          Keys.onReturnPressed: selector.playRequested(slot.index + 1)
          Keys.onSpacePressed: selector.playRequested(slot.index + 1)

          Text {
            anchors.fill: parent
            anchors.margins: 9
            text: (slot.index + 1) + "  " + (slot.preset.name || "Empty slot")
            textFormat: Text.PlainText
            color: selector.foreground
            font.family: selector.fontFamily
            font.pixelSize: selector.fontSize
            font.bold: slot.preset.active === true
            verticalAlignment: Text.AlignVCenter
            elide: Text.ElideRight
          }

          ToolTip.visible: mouse.containsMouse && !mouse.drag.active
          ToolTip.delay: 500
          ToolTip.text: slot.preset.name || "Select a station and press H in Subweazl"

          MouseArea {
            id: mouse
            anchors.fill: parent
            hoverEnabled: true
            cursorShape: drag.active ? Qt.ClosedHandCursor : Qt.PointingHandCursor
            drag.target: tile
            drag.axis: Drag.XAndYAxis
            property bool moved: false
            onPressed: { moved = false; selector.draggedSlot = slot.index; tile.forceActiveFocus() }
            onPositionChanged: if (drag.active) moved = true
            onReleased: {
              if (moved) tile.Drag.drop()
              selector.draggedSlot = -1
              tile.x = 0
              tile.y = 0
            }
            onCanceled: { tile.Drag.cancel(); selector.draggedSlot = -1; tile.x = 0; tile.y = 0 }
            onClicked: if (!moved) selector.playRequested(slot.index + 1)
          }
        }
      }
    }
  }
}
