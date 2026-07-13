cmake_minimum_required(VERSION 3.16)
project(awss3 CXX)

include(CheckIncludeFileCXX)

check_include_file_cxx(any HAS_ANY)
check_include_file_cxx(string_view HAS_STRING_VIEW)
check_include_file_cxx(coroutine HAS_COROUTINE)
if (NOT "${CMAKE_CXX_STANDARD}" STREQUAL "")
    # Do nothing
elseif (HAS_ANY AND HAS_STRING_VIEW AND HAS_COROUTINE)
    set(CMAKE_CXX_STANDARD 20)
elseif (HAS_ANY AND HAS_STRING_VIEW)
    set(CMAKE_CXX_STANDARD 17)
else ()
    set(CMAKE_CXX_STANDARD 14)
endif ()

set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

# ##############################################################################
# vcpkg toolchain is wired in by the bootstrap helper (see scripts/build.sh) or
# by passing -DCMAKE_TOOLCHAIN_FILE=... to cmake. We just locate the packages.
# ##############################################################################
find_package(Drogon CONFIG REQUIRED)
find_package(AWSSDK CONFIG REQUIRED COMPONENTS s3)

# ##############################################################################
# Source layout
# ##############################################################################
add_executable(${PROJECT_NAME} main.cc)

aux_source_directory(controllers CTL_SRC)
aux_source_directory(filters FILTER_SRC)
aux_source_directory(plugins PLUGIN_SRC)
aux_source_directory(models MODEL_SRC)

drogon_create_views(${PROJECT_NAME} ${CMAKE_CURRENT_SOURCE_DIR}/views
                    ${CMAKE_CURRENT_BINARY_DIR})

target_include_directories(${PROJECT_NAME}
                           PRIVATE ${CMAKE_CURRENT_SOURCE_DIR}
                                   ${CMAKE_CURRENT_SOURCE_DIR}/models)
target_sources(${PROJECT_NAME}
               PRIVATE
               ${CTL_SRC}
               ${FILTER_SRC}
               ${PLUGIN_SRC}
               ${MODEL_SRC})

target_link_libraries(${PROJECT_NAME} PRIVATE
                      Drogon::Drogon
                      aws-cpp-sdk-s3)

# ##############################################################################
# Standard / spec diagnostics
# ##############################################################################
if (CMAKE_CXX_STANDARD LESS 17)
    message(FATAL_ERROR "c++17 or higher is required")
elseif (CMAKE_CXX_STANDARD LESS 20)
    message(STATUS "use c++17")
else ()
    message(STATUS "use c++20")
endif ()

# ##############################################################################
# Tests
# ##############################################################################
option(AWSS3_BUILD_TESTS "Build awss3 unit tests" ON)
if (AWSS3_BUILD_TESTS)
    enable_testing()
    add_subdirectory(test)
endif ()
