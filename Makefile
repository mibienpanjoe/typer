GO      ?= $(shell command -v go 2>/dev/null || echo /usr/local/go/bin/go)
PREFIX  ?= $(HOME)/.local
BINDIR  ?= $(PREFIX)/bin

.PHONY: help test build install uninstall

help:
	@echo "make test       tests"
	@echo "make build      binaire ./typer"
	@echo "make install    $(BINDIR)/typer  (PATH)"
	@echo "make uninstall  retire $(BINDIR)/typer"

test:
	$(GO) test ./...

build:
	$(GO) build -o typer ./cmd/typer

install:
	mkdir -p "$(BINDIR)"
	$(GO) build -o "$(BINDIR)/typer" ./cmd/typer
	@echo "installé : $(BINDIR)/typer"
	@found="$$(command -v typer 2>/dev/null || true)"; \
	if [ "$$found" = "$(BINDIR)/typer" ]; then \
		echo "commande : $$found"; \
	elif [ -n "$$found" ]; then \
		echo "commande actuelle : $$found"; \
		echo "ajoute $(BINDIR) au début du PATH pour utiliser cette installation"; \
	else \
		echo "ajoute $(BINDIR) à ton PATH si typer n'est pas trouvé"; \
	fi

uninstall:
	rm -f "$(BINDIR)/typer"
	@echo "retiré : $(BINDIR)/typer"
