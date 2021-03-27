package main

import (
	//"time"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/rrrkren/topshot-sales/topshot"

	"github.com/onflow/flow-go-sdk/client"
	"google.golang.org/grpc"
)

var gameData topshot.Data
var previousId uint64

func handleErr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func main() {

	gameData = topshot.LoadGameData()
	// connect to flow
	flowClient, err := client.New("access.mainnet.nodes.onflow.org:9000", grpc.WithInsecure())
	handleErr(err)
	err = flowClient.Ping(context.Background())
	handleErr(err)

	// Run a bigger fetch block the first time, to check more blocks in the past:
	// latestBlock, err := flowClient.GetLatestBlock(context.Background(), true)
	// handleErr(err)

	//fetchBlocks(flowClient, int64(latestBlock.Height - 50), int64(latestBlock.Height), "A.c1e4f4f4c4257510.Market.MomentListed")

	for {
		// fetch latest block
		latestBlock, err := flowClient.GetLatestBlock(context.Background(), false)
		handleErr(err)
		//fmt.Println("current height: ", latestBlock.Height)

		blockSize := 10

		//start := time.Now()
		for i := 0; i < blockSize; i += blockSize {
			//fmt.Println("current block: ", int64(latestBlock.Height) - int64(i))

			fetchBlocks(flowClient, int64(latestBlock.Height)-int64(i)-int64(blockSize), int64(latestBlock.Height)-int64(i), "A.c1e4f4f4c4257510.Market.MomentListed")

			//fetchBlocks(flowClient, int64(latestBlock.Height) - int64(i) - int64(blockSize), int64(latestBlock.Height) - int64(i), "A.c1e4f4f4c4257510.Market.MomentPriceChanged")
		}
		// elapsed := time.Since(start)
		// fmt.Println("Fetch block took %s", elapsed)
		fmt.Print(".")
	}

}

func fetchBlocks(flowClient *client.Client, startBlock int64, endBlock int64, typeStr string) {
	// fetch block events of topshot Market.MomentListed/PriceChanged events for the past 1000 blocks
	blockEvents, err := flowClient.GetEventsForHeightRange(context.Background(), client.EventRangeQuery{
		Type:        typeStr,
		StartHeight: uint64(startBlock),
		EndHeight:   uint64(endBlock),
	})
	handleErr(err)

	for _, blockEvent := range blockEvents {

		for _, sellerEvent := range blockEvent.Events {
			// loop through the Market.MomentListed/PriceChanged events in this blockEvent
			// fmt.Println(sellerEvent.Value)
			e := topshot.MomentListed(sellerEvent.Value)

			eventId := e.Id()

			if previousId == eventId {
				continue
			}

			saleMoment, err := topshot.GetSaleMomentFromOwnerAtBlock(flowClient, blockEvent.Height, *e.Seller(), eventId)
			handleErr(err)

			if shouldPrintPlayer(e, saleMoment) {
				printPlayer(saleMoment, true)

				previousId = eventId
			}

		}
	}
}

func shouldPrintPlayer(moment topshot.MomentListed, sale *topshot.SaleMoment) bool {
	/**
	* Sale:
	* 	- NumMoments
	* 	- SerialNumber
	* 	- JerseyNumber
	*		- SetID
	*
	* Moment:
	*		- Price
	 */

	/**
	* Set IDs:
	* 	- 2: Base Set (Series 1)
	*		- 5: Metallic Gold LE
	*		- 6: Early Adopters
	*   - 22: Got Game
	*		- 26: Base Set (Series 2)
	* 	- 29: Metallic Gold LE
	* 	- 32: Cool Cats
	*		- 33: The Gift
	*		- 34: Seeing Stars
	* 	- 36: 2021 All-Star Game
	 */

	if sale == nil {
		return false
	}

	switch sale.SetID() {
	case
		2,
		5,
		26,
		32,
		33,
		34,
		36:
		return false
	}

	if sale.SetID() != 26 {
		fmt.Println("SetID: ", sale.SetID())
		fmt.Println("Set name: ", sale.SetName())
		return true
	}

	// if moment.Price() < 10 {
	// 	return true
	// }

	return false
}

func isMomentVeryRare(sale *topshot.SaleMoment) bool {
	if sale.NumMoments() <= 3000 || sale.SerialNumber() == sale.JerseyNumber() {
		return true
	}
	return false
}

func isMomentRare(sale *topshot.SaleMoment) bool {
	return sale.NumMoments() <= 15000
}

func isMomentSerialLow(sale *topshot.SaleMoment) bool {
	return sale.SerialNumber() <= 200
}

func printPlayer(saleMoment *topshot.SaleMoment, printURL bool) {
	c := color.New(color.FgWhite)
	if isMomentVeryRare(saleMoment) {
		c = c.Add(color.FgRed)
	}
	if isMomentRare(saleMoment) {
		c = c.Add(color.FgGreen)
	}
	if isMomentSerialLow(saleMoment) {
		c = c.Add(color.Bold)
	}

	c.Println("")
	c.Println(saleMoment, "\tPrice: ", fmt.Sprintf("%.0f", saleMoment.Price()))

	if printURL {
		url := getPlayerURL(saleMoment)

		c.Println(url)

		// openURLOnChrome(url)

		shoutSale(saleMoment)
	}

	fmt.Println("")
}

func getMomentInfoFromPlayerID(playerId int, momentsCount uint32, price float64) []byte {
	playerIdStr := strconv.Itoa(playerId)
	momentsCountStr := strconv.Itoa(int(momentsCount))
	//priceStr := fmt.Sprintf("%.0f", price)

	queryData := "{\"operationName\":\"SearchMomentListingsDefault\",\"variables\":{\"byPrice\":{\"min\":null,\"max\":\"1000" + "\"},\"byPower\":{\"min\":null,\"max\":null},\"bySerialNumber\":{\"min\":null,\"max\":\"" + momentsCountStr + "\"},\"byGameDate\":{\"start\":null,\"end\":null},\"byCreatedAt\":{\"start\":null,\"end\":null},\"byPrimaryPlayerPosition\":[],\"bySets\":[],\"bySeries\":[],\"bySetVisuals\":[],\"byPlayStyle\":[],\"bySkill\":[],\"byPlayers\":[\"" + playerIdStr + "\"],\"byTagNames\":[],\"byTeams\":[],\"byListingType\":[\"BY_USERS\"],\"searchInput\":{\"pagination\":{\"cursor\":\"\",\"direction\":\"RIGHT\",\"limit\":12}},\"orderBy\":\"UPDATED_AT_DESC\"},\"query\":\"query SearchMomentListingsDefault($byPlayers: [ID], $byTagNames: [String!], $byTeams: [ID], $byPrice: PriceRangeFilterInput, $orderBy: MomentListingSortType, $byGameDate: DateRangeFilterInput, $byCreatedAt: DateRangeFilterInput, $byListingType: [MomentListingType], $bySets: [ID], $bySeries: [ID], $bySetVisuals: [VisualIdType], $byPrimaryPlayerPosition: [PlayerPosition], $bySerialNumber: IntegerRangeFilterInput, $searchInput: BaseSearchInput!, $userDapperID: ID) {\n  searchMomentListings(input: {filters: {byPlayers: $byPlayers, byTagNames: $byTagNames, byGameDate: $byGameDate, byCreatedAt: $byCreatedAt, byTeams: $byTeams, byPrice: $byPrice, byListingType: $byListingType, byPrimaryPlayerPosition: $byPrimaryPlayerPosition, bySets: $bySets, bySeries: $bySeries, bySetVisuals: $bySetVisuals, bySerialNumber: $bySerialNumber}, sortBy: $orderBy, searchInput: $searchInput, userDapperID: $userDapperID}) {\n    data {\n      filters {\n        byPlayers\n        byTagNames\n        byTeams\n        byPrimaryPlayerPosition\n        byGameDate {\n          start\n          end\n          __typename\n        }\n        byCreatedAt {\n          start\n          end\n          __typename\n        }\n        byPrice {\n          min\n          max\n          __typename\n        }\n        bySerialNumber {\n          min\n          max\n          __typename\n        }\n        bySets\n        bySeries\n        bySetVisuals\n        __typename\n      }\n      searchSummary {\n        count {\n          count\n          __typename\n        }\n        pagination {\n          leftCursor\n          rightCursor\n          __typename\n        }\n        data {\n          ... on MomentListings {\n            size\n            data {\n              ... on MomentListing {\n                id\n                version\n                circulationCount\n                flowRetired\n                set {\n                  id\n                  flowName\n                  setVisualId\n                  flowSeriesNumber\n                  __typename\n                }\n                play {\n                  description\n                  id\n                  stats {\n                    playerName\n                    dateOfMoment\n                    playCategory\n                    teamAtMomentNbaId\n                    teamAtMoment\n                    __typename\n                  }\n                  __typename\n                }\n                assetPathPrefix\n                priceRange {\n                  min\n                  max\n                  __typename\n                }\n                momentListingCount\n                listingType\n                userOwnedSetPlayCount\n                __typename\n              }\n              __typename\n            }\n            __typename\n          }\n          __typename\n        }\n        __typename\n      }\n      __typename\n    }\n    __typename\n  }\n}\n\"}"
	//queryData = strings.ReplaceAll(queryData, "\\n", "\n")
	queryData = strings.Replace(queryData, "\n", `\n`, -1)

	args := []string{"--header", "Content-Type: application/json", "--data", queryData, "https://api.nbatopshot.com/marketplace/graphql?SearchMomentListingsDefault"}
	c := exec.Command("curl", args...)

	output, _ := c.Output()

	return output
}

func getPlayerURL(saleMoment *topshot.SaleMoment) string {
	playData := saleMoment.Play()
	playerIdStr := gameData.GetPlayerIDForName(playData["FullName"])

	playerId, _ := strconv.Atoi(playerIdStr)

	//fmt.Println("https://www.nbatopshot.com/search?byPlayers="+gameData.GetPlayerIDForName(playData["FullName"]))
	jsonBytes := getMomentInfoFromPlayerID(playerId, saleMoment.NumMoments(), saleMoment.Price())

	var postData topshot.POSTData
	err := json.Unmarshal(jsonBytes, &postData)
	if err != nil {
		fmt.Println("error:", err)
	}

	momentListings := postData.GetMomentListings()
	momentCount := len(momentListings)

	if momentCount > 1 {
		//args := []string{"Warning!","Found", "moment", "with", "more", "than", "1", "listing"}
		//shoutStrings(args)

		for i := 0; i < momentCount; i += 1 {
			// Each value is an interface{} type, that is type asserted as a string
			moment := momentListings[i]

			if (int(moment.Count) == int(saleMoment.NumMoments())) && (moment.SetData.Name == saleMoment.SetName()) {
				return "https://www.nbatopshot.com/listings/p2p/" + moment.GetURLHash()
			}
		}

		fmt.Println("too many moments, no url found.")
		return ""

	} else if momentCount == 1 {
		moment := momentListings[0]
		return "https://www.nbatopshot.com/listings/p2p/" + moment.GetURLHash()
	}

	fmt.Println("no moments found:", momentCount)
	return ""
}

func getLowestAsk(url string) int {
	// Scrape from player url
	return 0
}

func openURLOnChrome(url string) {
	if url == "" {
		return
	}
	if runtime.GOOS == "windows" {
		args := []string{url}
		c := exec.Command("C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe", args...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stdout
		c.Run()

	} else {
		args := []string{"--new", "-a", "Google Chrome", "--args", url}
		c := exec.Command("open", args...)
		c.Stdout = os.Stdout
		c.Run()
	}
}

func shoutSale(saleMoment *topshot.SaleMoment) {
	if runtime.GOOS == "windows" {
		return
	}

	serialStr := strconv.FormatUint(uint64(saleMoment.SerialNumber()), 10)
	totalStr := strconv.FormatUint(uint64(saleMoment.NumMoments()), 10)
	priceStr := fmt.Sprintf("%.0f", saleMoment.Price()) + "$"

	args := []string{"serial", serialStr, "of", totalStr, "price", priceStr}
	shoutStrings(args)
}

func shoutStrings(args []string) {
	c := exec.Command("say", args...)
	c.Stdout = os.Stdout
	c.Run()
}
