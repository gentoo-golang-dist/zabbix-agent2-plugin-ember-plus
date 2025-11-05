# libember_slim build script

LIBEMBER_BUILD=$(LIBEMBER)/build
INCLUDES=-I$(LIBEMBER)/libember_slim/Source

CGO_CFLAGS += $(INCLUDES)
CGO_LDFLAGS += -L$(LIBEMBER_BUILD)

CMAKE=cmake

ifeq ($(OS),Windows_NT)
CC = gcc
AR = ar
CMAKE_OPT=-G "MinGW Makefiles"
endif

ifeq ($(OS),Windows_NT)
RM = del /Q
else
RM = rm -f
endif

$(LIBEMBER): libember_slim-static.a libember_slim_cb.a

$(LIBEMBER_BUILD):
ifeq ($(OS),Windows_NT)
	if not exist "$(LIBEMBER_BUILD)" mkdir "$(LIBEMBER_BUILD)"
else
	mkdir -p "$(LIBEMBER_BUILD)"
endif


$(LIBEMBER_BUILD)/libember_slim_cb.o: $(LIBEMBER)/libember_slim_cb.c | $(LIBEMBER_BUILD) 
	$(CC) $(INCLUDES) $(CPPFLAGS) $(CFLAGS) -o "$@" -c "$^"

$(LIBEMBER_BUILD)/libember_slim_cb.a: $(LIBEMBER_BUILD)/libember_slim_cb.o
	$(RM) "$@"
	$(AR) crs "$@" "$^"

libember_slim_cb.a: $(LIBEMBER_BUILD)/libember_slim_cb.a


$(LIBEMBER_BUILD)/Makefile: | $(LIBEMBER_BUILD)
	cd $(LIBEMBER_BUILD)
	$(CMAKE) $(LIBEMBER)/libember_slim $(CMAKE_OPT)

$(LIBEMBER_BUILD)/libember_slim-static.a: $(LIBEMBER_BUILD)/Makefile
	$(MAKE) -C $(LIBEMBER_BUILD)

libember_slim-static.a: $(LIBEMBER_BUILD)/libember_slim-static.a
