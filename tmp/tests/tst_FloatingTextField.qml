import QtQuick 2.15
import QtTest 1.2
import "../js"

TestCase {
    name: "FloatingTextFieldTest"
    width: 400
    height: 400
    visible: true
    when: windowShown

    // Instanciamos el componente a probar
    FloatingTextField {
        id: field
        anchors.centerIn: parent
        width: 200
        label: "Test Label"
    }

    function test_initialState() {
        compare(field.text, "", "El texto inicial debería estar vacío")
        compare(field.label, "Test Label", "La etiqueta debería coincidir")
        verify(!field.activeFocus, "No debería tener foco al inicio")
    }

    function test_input() {
        // Simulamos dar foco y escribir
        field.forceActiveFocus()
        verify(field.activeFocus, "El campo debería tener foco")
        keyClick(Qt.Key_A)
        compare(field.text, "a", "El texto debería ser 'a' después de escribir")
    }
}