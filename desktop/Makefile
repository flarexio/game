TARGET = desktop

SRC = desktop.cpp

CXX = clang++ 
CXXFLAGS = -std=c++11 -Wall -O2

INCLUDES = 

LDFLAGS = 

LIBS = -ld3d11 -ldxgi

$(TARGET):
	$(CXX) $(CXXFLAGS) $(INCLUDES) $(LDFLAGS) -o $(TARGET) $(SRC) $(LIBS)

clean:
	rm -f $(TARGET)
