#!/usr/bin/env bash

BEDROCK_TAGS_REMOTE="$1"
VERSION="$2"

if [ -z "$VERSION" ]; then
	echo "You must specify a version."
	exit 0
fi

FIRST_CHAR=$(printf '%s' "$VERSION" | cut -c1)
if [ "$FIRST_CHAR" != "v" ]; then
	echo "Tag must start with v."
	exit 0
fi

git tag "op-bindings/$VERSION"
git tag "op-service/$VERSION"
git push $BEDROCK_TAGS_REMOTE "op-bindings/$VERSION"
git push $BEDROCK_TAGS_REMOTE "op-service/$VERSION"

cd op-chain-ops
go get github.com/ethereum-optimism/optimism/op-bindings@$VERSION
go get github.com/ethereum-optimism/optimism/op-service@$VERSION
go mod tidy

git add .
git commit -am 'chore: Upgrade op-chain-ops dependencies'

git tag "op-chain-ops/$VERSION"
git push $BEDROCK_TAGS_REMOTE "op-chain-ops/$VERSION"

cd ../op-node
go get github.com/ethereum-optimism/optimism/op-bindings@$VERSION
go get github.com/ethereum-optimism/optimism/op-service@$VERSION
go get github.com/ethereum-optimism/optimism/op-chain-ops@$VERSION
go mod tidy

git add .
git commit -am 'chore: Upgrade op-node dependencies'
git push $BEDROCK_TAGS_REMOTE
git tag "op-node/$VERSION"
git push $BEDROCK_TAGS_REMOTE "op-node/$VERSION"

cd ../op-proposer
go get github.com/ethereum-optimism/optimism/op-bindings@$VERSION
go get github.com/ethereum-optimism/optimism/op-service@$VERSION
go get github.com/ethereum-optimism/optimism/op-node@$VERSION
go mod tidy

git add .
git commit -am 'chore: Upgrade op-proposer dependencies'
git push $BEDROCK_TAGS_REMOTE
git tag "op-proposer/$VERSION"
git push $BEDROCK_TAGS_REMOTE "op-proposer/$VERSION"

cd ../op-batcher
go get github.com/ethereum-optimism/optimism/op-bindings@$VERSION
go get github.com/ethereum-optimism/optimism/op-service@$VERSION
go get github.com/ethereum-optimism/optimism/op-node@$VERSION
go get github.com/ethereum-optimism/optimism/op-proposer@$VERSION
go mod tidy

git add .
git commit -am 'chore: Upgrade op-batcher dependencies'
git push $BEDROCK_TAGS_REMOTE
git tag "op-batcher/$VERSION"
git push $BEDROCK_TAGS_REMOTE "op-batcher/$VERSION"

cd ../op-e2e
go get github.com/ethereum-optimism/optimism/op-bindings@$VERSION
go get github.com/ethereum-optimism/optimism/op-service@$VERSION
go get github.com/ethereum-optimism/optimism/op-node@$VERSION
go get github.com/ethereum-optimism/optimism/op-proposer@$VERSION
go get github.com/ethereum-optimism/optimism/op-batcher@$VERSION
go mod tidy

git add .
git commit -am 'chore: Upgrade op-e2e dependencies'
git push $BEDROCK_TAGS_REMOTE
git tag "op-e2e/$VERSION"
git push $BEDROCK_TAGS_REMOTE "op-e2e/$VERSION"