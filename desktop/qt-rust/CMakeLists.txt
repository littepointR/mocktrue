
cmake_minimum_required(VERSION 3.24)

project(PortWeaveQtRust LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_AUTOMOC ON)
set(CMAKE_AUTORCC ON)
set(CMAKE_AUTOUIC ON)

find_package(Qt6 6.5 REQUIRED COMPONENTS Core Gui Qml Quick)

if(NOT DEFINED PORTWEAVE_RUST_CARGO)
    find_program(PORTWEAVE_RUST_CARGO cargo REQUIRED)
endif()

set(PORTWEAVE_QT_RUST_RUST_DIR ${CMAKE_CURRENT_SOURCE_DIR}/rust)
set(PORTWEAVE_QT_RUST_RUST_TARGET_DIR ${CMAKE_BINARY_DIR}/rust-target)
set(PORTWEAVE_QT_RUST_RUST_PROFILE debug)
if(CMAKE_BUILD_TYPE MATCHES "^[Rr]elease$")
    set(PORTWEAVE_QT_RUST_RUST_PROFILE release)
endif()

if(WIN32)
    set(PORTWEAVE_QT_RUST_RUST_LIB ${PORTWEAVE_QT_RUST_RUST_TARGET_DIR}/${PORTWEAVE_QT_RUST_RUST_PROFILE}/portweave_qt_rust.lib)
else()
    set(PORTWEAVE_QT_RUST_RUST_LIB ${PORTWEAVE_QT_RUST_RUST_TARGET_DIR}/${PORTWEAVE_QT_RUST_RUST_PROFILE}/libportweave_qt_rust.a)
endif()

file(GLOB_RECURSE PORTWEAVE_QT_RUST_RUST_SOURCES CONFIGURE_DEPENDS
    ${PORTWEAVE_QT_RUST_RUST_DIR}/src/*.rs
    ${PORTWEAVE_QT_RUST_RUST_DIR}/Cargo.toml
)

add_custom_command(
    OUTPUT ${PORTWEAVE_QT_RUST_RUST_LIB}
    COMMAND ${CMAKE_COMMAND} -E env
            CARGO_TARGET_DIR=${PORTWEAVE_QT_RUST_RUST_TARGET_DIR}
            ${PORTWEAVE_RUST_CARGO} build --manifest-path ${PORTWEAVE_QT_RUST_RUST_DIR}/Cargo.toml
    WORKING_DIRECTORY ${PORTWEAVE_QT_RUST_RUST_DIR}
    DEPENDS ${PORTWEAVE_QT_RUST_RUST_SOURCES}
    COMMENT "Building PortWeave Qt/Rust static library"
    VERBATIM
)

add_custom_target(portweave_qt_rust_core ALL DEPENDS ${PORTWEAVE_QT_RUST_RUST_LIB})

add_library(portweave_qt_rust_static STATIC IMPORTED GLOBAL)
set_target_properties(portweave_qt_rust_static PROPERTIES
    IMPORTED_LOCATION ${PORTWEAVE_QT_RUST_RUST_LIB}
)
add_dependencies(portweave_qt_rust_static portweave_qt_rust_core)

add_executable(PortWeaveQtRust
    app/main.cpp
    app/AppBridge.cpp
    app/WindowLifecycleProbe.cpp
)

target_include_directories(PortWeaveQtRust PRIVATE ${CMAKE_CURRENT_SOURCE_DIR}/app)
target_link_libraries(PortWeaveQtRust PRIVATE
    Qt6::Core
    Qt6::Gui
    Qt6::Qml
    Qt6::Quick
    portweave_qt_rust_static
)
add_dependencies(PortWeaveQtRust portweave_qt_rust_core)

add_custom_command(TARGET PortWeaveQtRust POST_BUILD
    COMMAND ${CMAKE_COMMAND} -E copy_directory
            ${CMAKE_CURRENT_SOURCE_DIR}/qml
            $<TARGET_FILE_DIR:PortWeaveQtRust>/qml
)
