cmake_minimum_required(VERSION 3.16)
project(awss3_test CXX)

add_executable(${PROJECT_NAME} test_main.cc)
target_link_libraries(${PROJECT_NAME} PRIVATE Drogon::Drogon)

enable_testing()
ParseAndAddDrogonTests(${PROJECT_NAME})
