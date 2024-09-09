# ENV
monorepo_root := "../../../.."
deploy_config_path := "../../deploy-config"
tmp := $(shell mktemp)

# Help menu
.PHONY: help
help: # Show help for each of the Makefile recipes.
	@grep -E '^[a-zA-Z0-9 -]+:.*#'  Makefile | sort | while read -r l; do printf "\033[1;32m$$(echo $$l | cut -f 1 -d':')\033[00m:$$(echo $$l | cut -f 2- -d'#')\n"; done

.PHONY: cannon-prestate
cannon-prestate: # Generate the cannon prestate, and tar the `op-program` + `cannon` binaries + prestate data used to generate it.
	cd $(monorepo_root) && make reproducible-prestate
	@mkdir -p prestate-artifacts
	@cp -r $(monorepo_root)/cannon/bin/** prestate-artifacts/
	@cp -r $(monorepo_root)/op-program/bin/** prestate-artifacts/
	tar -czvf prestate-artifacts.tar.gz prestate-artifacts
	@jq ".faultGameAbsolutePrestate = $$(cat prestate-artifacts/prestate-proof.json | jq .pre)" "$(deploy_config_path)/$(chain).json" > $(tmp) && mv $(tmp) "$(deploy_config_path)/$(chain).json"
	@echo "-------------------------------------------------------------------------------------------------------------"
	@echo "-> Archive of prestate artifacts available at prestate-artifacts.tar.gz"
	@echo "-> Set the absolute prestate to $$(cat prestate-artifacts/prestate-proof.json | jq .pre) in the $(chain) deploy config."
	@rm -rf prestate-artifacts

.PHONY: deploy-fresh
deploy-fresh: cannon-prestate # Deploy a fresh version of the FPAC contracts. Pass `--broadcast` to send to the network.
	forge script FPACOPS.s.sol --sig "deployFPAC(address,address,address)" $(proxy-admin) $(system-owner-safe) $(superchain-config-proxy) --chain $(chain) -vvv $(args)

# TODO: Convert this whole file to a justfile
.PHONY: deploy-upgrade
deploy-upgrade: cannon-prestate # Deploy upgraded FP contracts. Pass `--broadcast` to send to the network.
	forge script FPACOPS2.s.sol --sig "deployFPAC2(address,address,address,address,address)" $(proxy-admin) $(system-owner-safe) $(superchain-config-proxy) $(dispute-game-factory-proxy) $(anchor-state-registry-proxy) --chain $(chain) -vvv $(args)
