import QtQuick
import QtTest

TestCase {
    name: "ExampleTests"

    function test_addition() {
        compare(1 + 1, 2, "1 + 1 should be 2");
    }

    function test_stringComparison() {
        verify("hello" === "hello", "Strings should match");
    }
}
