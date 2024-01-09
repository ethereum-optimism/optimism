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
	cd $(monorepo_root) && make cannon-prestate
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
deploy-fresh: # Deploy a fresh version of the FPAC contracts. Pass `--broadcast` to send to the network.
	forge script FPACOPS.sol --sig "deployFPAC()" --chain $(chain) -vvv $(args)

.PHONY: upgrade-game-impl
upgrade-game-impl: # Upgrades a dispute game implementation in the deployed `DisputeGameFactory` contract for the configured chain. Pass `--broadcast` to send to the network.
	forge script FPACOPS.sol --sig "upgradeGameImpl(address,address)" --chain $(chain) -vvv $(dgf) $(vm) $(args)

.PHONY: update-init-bond
update-init-bond: # Upgrades a dispute game's init bond in the deployed `DisputeGameFactory` contract for the configured chain. Pass `--broadcast` to send to the network.
	forge script FPACOPS.sol --sig "updateInitBond(address,uint8,uint256)" --chain $(chain) -vvv $(dgf) $(game-type) $(new-init-bond) $(args)
