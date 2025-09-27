# jna-nuh-yoh-guh
ज्ञान योग

Currnt Task:
The Consolidation Workflow (Step-by-Step)
This process should be a distinct workflow, perhaps triggered by a new API endpoint like POST /api/v1/consolidate. It runs after one or more narratives have been analyzed and their raw, unconsolidated graphs have been created.

Here are the steps:

Step 1: Generate Embeddings
For every new node where consolidation = false:

Combine its most descriptive text (e.g., node.name + " " + node.description).

Send this text to an embedding model API.

Store the resulting vector (e.g., a list of 768 numbers) with the node, either temporarily in your Go application's memory or as a new property on the node itself.

Step 2: Match Nodes
Iterate through each of your unconsolidated nodes. For each one:

Compare its vector against the vectors of all consolidated nodes of the same type (e.g., compare a new Stock only against existing consolidated Stocks).

Calculate the cosine similarity for each pair.

If the similarity score is above a certain threshold (e.g., > 0.9), you've found a match! This threshold is tunable; a higher value means you require a closer semantic match.

Step 3: Merge and Update 📈
This is the core database transaction. For each unconsolidated node:

If a match is found:

Don't create a new node. You'll use the existing consolidated one.

Create a temporary map in your Go code to link the old ID to the new one (e.g., id_map["unconsolidated_stock_123"] = "consolidated_stock_abc").

Run a Cypher query to MATCH the existing consolidated node and SET its consolidation_score = consolidation_score + 1.

If no match is found:

This is a genuinely new concept. "Promote" the node by running a query to SET its consolidation = true and consolidation_score = 1.

After all nodes are processed, use your id_map to re-wire the relationships. For each unconsolidated relationship, look up the new consolidated IDs for its start and end nodes.

Check if a relationship of the same type already exists between these two consolidated nodes.

If it does, simply increment its consolidation_score.

If it doesn't, create the new relationship and set its consolidation = true and consolidation_score = 1.

Finally, delete all the unconsolidated nodes that were successfully merged.

Handling Causal Links 🔗
As you rightly pointed out, CausalLink is special. It represents uncertainty, so a simple score doesn't make sense. Here’s a powerful way to handle them:

Re-point, Don't Score: When you consolidate nodes, use your id_map to re-point the CausalLink relationships to the new consolidated nodes. Do not add a consolidation_score.

Aggregate Evidence: If re-pointing a CausalLink results in a duplicate (i.e., another CausalLink already exists between the same two consolidated nodes), do not simply discard it. Instead, append its question property to a list on the existing relationship.

Example:

Initial Link: (Energy Level)-[:CAUSAL_LINK {question: "Does low energy hurt focus?"}]->(Focus)

A new narrative creates a link that gets merged: (Vigor)-[:CAUSAL_LINK {question: "I can't focus when I'm tired."}]->(Concentration)

After consolidation, you would have a single, richer relationship:
(Consolidated Energy)-[:CAUSAL_LINK {questions: ["Does low energy hurt focus?", "I can't focus when I'm tired."]}]->(Consolidated Focus)

This turns your graph from a collection of individual uncertainties into an aggregated body of evidence, which is perfect for the "resolution logic" you plan to build later.