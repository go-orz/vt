package main

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/dushixiang/vt-go"
)

var data = `{"version": 2, "width": 81, "height": 20}
[0.006808, "o", "> "]
[0.8880290000000001, "o", "#"]
[0.9601360000000001, "o", " "]
[1.160145, "o", "W"]
[1.343879, "o", "e"]
[1.53569, "o", "l"]
[1.631953, "o", "c"]
[1.735633, "o", "o"]
[1.808077, "o", "m"]
[1.840147, "o", "e"]
[1.927969, "o", " "]
[2.047975, "o", "t"]
[2.168139, "o", "o"]
[2.19972, "o", " "]
[2.46363, "o", "a"]
[2.5279540000000003, "o", "s"]
[2.711625, "o", "c"]
[2.847993, "o", "i"]
[2.991855, "o", "i"]
[3.199762, "o", "n"]
[3.295793, "o", "e"]
[3.415827, "o", "m"]
[3.49555, "o", "a"]
[3.8160540000000003, "o", "!"]
[4.3999690000000011, "o", "\r\n"]
[4.400176, "o", "> "]
[5.863699, "o", "#"]
[5.943767, "o", " "]
[6.191718, "o", "S"]
[6.447508, "o", "e"]
[6.623432, "o", "e"]
[6.751711, "o", " "]
[6.991612, "o", "h"]
[7.055641, "o", "o"]
[7.120098, "o", "w"]
[7.207876, "o", " "]
[7.455603, "o", "e"]
[7.703421, "o", "a"]
[7.756741, "o", "s"]
[7.967907, "o", "y"]
[8.135836000000001, "o", " "]
[8.407537000000001, "o", "i"]
[8.511744000000002, "o", "t"]
[8.647857000000002, "o", " "]
[8.872389000000002, "o", "i"]
[8.991854000000002, "o", "s"]
[9.055458000000002, "o", " "]
[9.348641000000002, "o", "t"]
[9.463529000000003, "o", "o"]
[9.527540000000004, "o", " "]
[9.663558000000004, "o", "r"]
[9.727847000000004, "o", "e"]
[9.911423000000005, "o", "c"]
[10.087556000000005, "o", "o"]
[10.167946000000004, "o", "r"]
[10.375745000000004, "o", "d"]
[10.455546000000004, "o", " "]
[10.679717000000004, "o", "a"]
[10.751638000000003, "o", " "]
[10.935322000000003, "o", "t"]
[11.119585000000002, "o", "e"]
[11.191345000000002, "o", "r"]
[11.319788000000003, "o", "m"]
[11.503195000000003, "o", "i"]
[11.567132000000003, "o", "n"]
[11.671429000000003, "o", "a"]
[11.863684000000003, "o", "l"]
[11.983449000000002, "o", " "]
[12.135647000000002, "o", "s"]
[12.311402000000003, "o", "e"]
[12.487300000000003, "o", "s"]
[12.623363000000003, "o", "s"]
[12.751410000000003, "o", "i"]
[12.815376000000004, "o", "o"]
[12.975492000000004, "o", "n"]
[13.415790000000005, "o", "\r\n"]
[13.415876000000004, "o", "> "]
[15.415876000000004, "o", "#"]
[15.511647000000004, "o", " "]
[15.767655000000003, "o", "F"]
[15.959906000000004, "o", "i"]
[16.023877000000002, "o", "r"]
[16.223933000000002, "o", "s"]
[16.383972000000004, "o", "t"]
[16.488035000000004, "o", " "]
[16.839736000000002, "o", "i"]
[16.872004, "o", "n"]
[16.99177, "o", "s"]
[17.144098, "o", "t"]
[17.232162, "o", "a"]
[17.607730999999998, "o", "l"]
[17.751799, "o", "l"]
[17.832068999999997, "o", " "]
[17.984004999999996, "o", "t"]
[18.143999999999995, "o", "h"]
[18.279733999999994, "o", "e"]
[18.383875999999994, "o", " "]
[18.751761999999992, "o", "a"]
[18.831954999999994, "o", "s"]
[19.071480999999995, "o", "c"]
[19.231423999999993, "o", "i"]
[19.367812999999995, "o", "i"]
[19.871739999999996, "o", "n"]
[19.967535999999996, "o", "e"]
[20.087669999999996, "o", "m"]
[20.191665999999994, "o", "a"]
[20.437538999999994, "o", " "]
[20.687428999999995, "o", "r"]
[20.767609999999994, "o", "e"]
[20.935535999999995, "o", "c"]
[21.127884999999996, "o", "o"]
[21.239638999999997, "o", "r"]
[21.447967, "o", "d"]
[21.695773, "o", "e"]
[21.759878, "o", "r"]
[22.247933, "o", "\r\n"]
[22.248017, "o", "> "]
[24.809024, "o", "b"]
[24.904177, "o", "r"]
[25.000196, "o", "e"]
[25.240239, "o", "w"]
[25.336285999999998, "o", " "]
[25.688201999999997, "o", "i"]
[25.735673999999996, "o", "n"]
[25.776016999999996, "o", "s"]
[25.984065999999995, "o", "t"]
[26.144118999999996, "o", "a"]
[26.255835999999995, "o", "l"]
[26.408125999999996, "o", "l"]
[26.472132999999996, "o", " "]
[26.672158999999997, "o", "a"]
[26.736117999999998, "o", "s"]
[26.856170999999996, "o", "c"]
[27.024136999999996, "o", "i"]
[27.176147999999998, "o", "i"]
[27.432059999999996, "o", "n"]
[27.520082999999996, "o", "e"]
[27.648048999999997, "o", "m"]
[27.768056999999995, "o", "a"]
[28.271421999999994, "o", "\r\r\n"]
[28.671421, "o", "\u001b[34m==>\u001b[0m \u001b[1mDownloading https://homebrew.bintray.com/bottles/asciinema-2.0.2_2.catalina.bottle.1.tar.gz\u001b[0m\r\n"]
[28.671422, "o", "\u001b[34m==>\u001b[0m \u001b[1mDownloading from https://akamai.bintray.com/4a/4ac59de631594cea60621b45d85214e39a90a0ba8ddf4eeec5cba34bd6145711\u001b[0m\r\n"]
[28.771422, "o", "\r##############                                                            19.7%"]
[28.871422, "o", "\r######################################################################## 100.0%\r\n"]
[28.971422, "o", "\u001b[34m==>\u001b[0m \u001b[1mPouring asciinema-2.0.2_2.catalina.bottle.1.tar.gz\u001b[0m\r\n"]
[29.176831, "o", "🍺  /usr/local/Cellar/asciinema/2.0.2_2: 613 files, 6.4MB\r\n"]
[29.22386299999999, "o", "> "]
[31.22386299999999, "o", "#"]
[31.30390099999999, "o", " "]
[31.535972999999988, "o", "N"]
[31.816291999999986, "o", "o"]
[31.920366999999988, "o", "w"]
[32.04028599999999, "o", " "]
[32.32034699999999, "o", "s"]
[32.52813899999999, "o", "t"]
[32.58385899999999, "o", "a"]
[32.70369099999999, "o", "r"]
[32.89591399999999, "o", "t"]
[33.00000899999999, "o", " "]
[33.18414799999999, "o", "r"]
[33.23214399999999, "o", "e"]
[33.43222899999999, "o", "c"]
[33.60871399999999, "o", "o"]
[33.76028499999999, "o", "r"]
[33.95218799999999, "o", "d"]
[34.14398599999999, "o", "i"]
[34.21607199999999, "o", "n"]
[34.37588099999999, "o", "g"]
[35.56782899999999, "o", "\r\n"]
[35.56791799999999, "o", "> "]
[37.56791799999999, "o", "a"]
[37.63171099999999, "o", "s"]
[37.74403699999999, "o", "c"]
[37.90368099999999, "o", "i"]
[38.05688699999999, "o", "i"]
[38.41622899999999, "o", "n"]
[38.51164999999999, "o", "e"]
[38.61612599999999, "o", "m"]
[38.767600999999985, "o", "a"]
[38.88750599999999, "o", " "]
[39.07965699999999, "o", "r"]
[39.15154599999999, "o", "e"]
[39.48814299999999, "o", "c"]
[40.104083999999986, "o", "\r\n"]
[40.11336099999998, "o", "\u001b[0;32masciinema: recording asciicast to /tmp/u52erylk-ascii.cast\u001b[0m\r\n\u001b[0;32masciinema: press <ctrl-d> or type \"exit\" when you're done\u001b[0m\r\n"]
[40.11959599999998, "o", "\u001b[?1034h"]
[40.11960399999998, "o", "bash-3.2$ "]
[42.11960399999998, "o", "#"]
[42.247433999999984, "o", " "]
[42.37526399999999, "o", "I"]
[42.50309399999999, "o", " "]
[42.76725499999999, "o", "a"]
[43.03920699999999, "o", "m"]
[43.19063699999999, "o", " "]
[43.43886599999999, "o", "i"]
[43.49048399999999, "o", "n"]
[43.55895599999999, "o", " "]
[43.73512599999999, "o", "a"]
[43.83888899999999, "o", " "]
[43.97460899999999, "o", "n"]
[44.10313499999999, "o", "e"]
[44.18286699999999, "o", "w"]
[44.247357999999984, "o", " "]
[44.45488899999999, "o", "s"]
[44.567081999999985, "o", "h"]
[44.670797999999984, "o", "e"]
[44.78286099999998, "o", "l"]
[44.94325099999998, "o", "l"]
[45.038657999999984, "o", " "]
[45.17514399999998, "o", "i"]
[45.238903999999984, "o", "n"]
[45.29510899999998, "o", "s"]
[45.49514399999998, "o", "t"]
[45.678938999999986, "o", "a"]
[45.83118899999999, "o", "n"]
[45.93505299999999, "o", "c"]
[46.09496999999999, "o", "e"]
[46.19871599999999, "o", " "]
[46.41504899999999, "o", "w"]
[46.51089199999999, "o", "h"]
[46.58294599999999, "o", "i"]
[46.686871999999994, "o", "c"]
[46.854735999999995, "o", "h"]
[46.974728, "o", " "]
[47.24703, "o", "i"]
[47.310535, "o", "s"]
[47.390708000000004, "o", " "]
[47.598771000000006, "o", "b"]
[47.67875800000001, "o", "e"]
[47.86311900000001, "o", "i"]
[47.935161000000015, "o", "n"]
[48.014595000000014, "o", "g"]
[48.10283300000001, "o", " "]
[48.27870100000001, "o", "r"]
[48.34252100000001, "o", "e"]
[48.518504000000014, "o", "c"]
[48.65462100000001, "o", "o"]
[48.74300400000001, "o", "r"]
[48.92703400000001, "o", "d"]
[49.11864400000001, "o", "e"]
[49.23067400000001, "o", "d"]
[49.32685800000001, "o", " "]
[49.50248600000001, "o", "n"]
[49.59879100000001, "o", "o"]
[49.694873000000015, "o", "w"]
[50.241591000000014, "o", "\r\r\n"]
[50.24180000000001, "o", "bash-3.2$ "]
[52.24180000000001, "o", "s"]
[52.39122100000001, "o", "h"]
[52.48717700000001, "o", "a"]
[52.71919000000001, "o", "1"]
[53.02299900000001, "o", "s"]
[53.143340000000016, "o", "u"]
[53.49506200000002, "o", "m"]
[54.61489500000002, "o", " "]
[54.894874000000016, "o", "/"]
[55.022925000000015, "o", "e"]
[55.239097000000015, "o", "t"]
[55.918849000000016, "o", "c"]
[56.399366000000015, "o", "/"]
[56.59112500000001, "o", "f"]
[56.999262000000016, "o", "*"]
[58.014676000000016, "o", " "]
[58.33468000000001, "o", "|"]
[58.456133000000015, "o", " "]
[58.74269800000002, "o", "t"]
[58.83832400000002, "o", "a"]
[59.23084400000002, "o", "i"]
[59.69470700000002, "o", "l"]
[59.846532000000025, "o", " "]
[60.718703000000026, "o", "-"]
[60.76693400000003, "o", "n"]
[60.918637000000025, "o", " "]
[61.31877000000002, "o", "1"]
[61.44702600000002, "o", "0"]
[61.89480200000002, "o", " "]
[62.31084800000002, "o", "|"]
[62.52659500000002, "o", " "]
[63.16684300000002, "o", "l"]
[63.334338000000024, "o", "o"]
[63.510549000000026, "o", "l"]
[63.59074700000003, "o", "c"]
[63.65734700000003, "o", "a"]
[64.55867100000003, "o", "t"]
[64.64018100000003, "o", " "]
[65.15842900000003, "o", "-"]
[65.39865200000003, "o", "F"]
[65.60664900000003, "o", " "]
[66.19066200000003, "o", "0"]
[66.27849900000002, "o", "."]
[66.41439100000002, "o", "3"]
[66.95827900000002, "o", "\r\r\n"]
[67.01345200000002, "o", "\u001b[38;5;184md\u001b[0m"]
[67.01349300000001, "o", "\u001b[38;5;184ma\u001b[0m"]
[67.01354100000002, "o", "\u001b[38;5;184m3\u001b[0m"]
[67.01354500000002, "o", "\u001b[38;5;214m9\u001b[0m"]
[67.01356900000002, "o", "\u001b[38;5;214ma\u001b[0m"]
[67.01358900000002, "o", "\u001b[38;5;214m3\u001b[0m"]
[67.01364300000003, "o", "\u001b[38;5;208me\u001b[0m"]
[67.01365700000002, "o", "\u001b[38;5;208me\u001b[0m"]
[67.01367900000002, "o", "\u001b[38;5;208m5\u001b[0m"]
[67.01369600000002, "o", "\u001b[38;5;208me\u001b[0m"]
[67.01372300000003, "o", "\u001b[38;5;203m6\u001b[0m"]
[67.01376300000003, "o", "\u001b[38;5;203mb\u001b[0m"]
[67.01381700000003, "o", "\u001b[38;5;203m4\u001b[0m"]
[67.01385300000003, "o", "\u001b[38;5;203mb\u001b[0m"]
[67.01388100000003, "o", "\u001b[38;5;198m0\u001b[0m"]
[67.01393100000003, "o", "\u001b[38;5;198md\u001b[0m"]
[67.01395700000003, "o", "\u001b[38;5;198m3\u001b[0m"]
[67.01398800000004, "o", "\u001b[38;5;199m2\u001b[0m"]
[67.01402200000004, "o", "\u001b[38;5;199m5\u001b[0m"]
[67.01406300000004, "o", "\u001b[38;5;199m5\u001b[0m"]
[67.01410700000004, "o", "\u001b[38;5;164mb\u001b[0m"]
[67.01413500000004, "o", "\u001b[38;5;164mf\u001b[0m"]
[67.01415400000003, "o", "\u001b[38;5;164me\u001b[0m"]
[67.01418800000003, "o", "\u001b[38;5;164mf\u001b[0m"]
[67.01422800000003, "o", "\u001b[38;5;129m9\u001b[0m"]
[67.01426200000003, "o", "\u001b[38;5;129m5\u001b[0m"]
[67.01428300000003, "o", "\u001b[38;5;129m6\u001b[0m"]
[67.01431100000003, "o", "\u001b[38;5;93m0\u001b[0m"]
[67.01434600000003, "o", "\u001b[38;5;93m1\u001b[0m"]
[67.01436100000004, "o", "\u001b[38;5;93m8\u001b[0m"]
[67.01439300000004, "o", "\u001b[38;5;93m9\u001b[0m"]
[67.01449000000004, "o", "\u001b[38;5;63m0\u001b[0m"]
[67.01453400000004, "o", "\u001b[38;5;63ma\u001b[0m"]
[67.01461600000005, "o", "\u001b[38;5;63mf\u001b[0m"]
[67.01462800000004, "o", "\u001b[38;5;63md\u001b[0m"]
[67.01465500000005, "o", "\u001b[38;5;33m8\u001b[0m"]
[67.01468000000004, "o", "\u001b[38;5;33m0\u001b[0m"]
[67.01473100000004, "o", "\u001b[38;5;33m7\u001b[0m"]
[67.01478700000004, "o", "\u001b[38;5;39m0\u001b[0m"]
[67.01485200000005, "o", "\u001b[38;5;39m9\u001b[0m"]
[67.01488600000005, "o", "\u001b[38;5;39m \u001b[0m"]
[67.01491300000005, "o", "\u001b[38;5;44m \u001b[0m"]
[67.01494200000005, "o", "\u001b[38;5;44m/\u001b[0m"]
[67.01495200000005, "o", "\u001b[38;5;44me\u001b[0m"]
[67.01496800000005, "o", "\u001b[38;5;44mt\u001b[0m"]
[67.01502100000005, "o", "\u001b[38;5;49mc\u001b[0m"]
[67.01504400000005, "o", "\u001b[38;5;49m/\u001b[0m"]
[67.01505600000004, "o", "\u001b[38;5;49mf\u001b[0m"]
[67.01508700000005, "o", "\u001b[38;5;48mi\u001b[0m"]
[67.01511100000005, "o", "\u001b[38;5;48mn\u001b[0m"]
[67.01515100000005, "o", "\u001b[38;5;48md\u001b[0m"]
[67.01518300000005, "o", "\u001b[38;5;84m.\u001b[0m"]
[67.01519600000005, "o", "\u001b[38;5;83mc\u001b[0m"]
[67.01522600000004, "o", "\u001b[38;5;83mo\u001b[0m"]
[67.01522900000005, "o", "\u001b[38;5;83md\u001b[0m"]
[67.01525600000005, "o", "\u001b[38;5;119me\u001b[0m"]
[67.01527900000005, "o", "\u001b[38;5;118ms\u001b[0m"]
[67.01528200000006, "o", "\r\r\n"]
[67.01535700000005, "o", "\u001b[38;5;214m8\u001b[0m"]
[67.01538000000005, "o", "\u001b[38;5;214m8\u001b[0m"]
[67.01542800000006, "o", "\u001b[38;5;214md\u001b[0m"]
[67.01545700000005, "o", "\u001b[38;5;208md\u001b[0m"]
[67.01549200000005, "o", "\u001b[38;5;208m3\u001b[0m"]
[67.01550400000005, "o", "\u001b[38;5;208me\u001b[0m"]
[67.01557700000005, "o", "\u001b[38;5;208ma\u001b[0m"]
[67.01558100000005, "o", "\u001b[38;5;203m7\u001b[0m"]
[67.01560000000005, "o", "\u001b[38;5;203mf\u001b[0m"]
[67.01563000000004, "o", "\u001b[38;5;203mf\u001b[0m"]
[67.01566600000004, "o", "\u001b[38;5;203mc\u001b[0m"]
[67.01569400000004, "o", "\u001b[38;5;198mb\u001b[0m"]
[67.01570600000004, "o", "\u001b[38;5;198mb\u001b[0m"]
[67.01572000000003, "o", "\u001b[38;5;198m9\u001b[0m"]
[67.01574600000004, "o", "\u001b[38;5;199m1\u001b[0m"]
[67.01577200000004, "o", "\u001b[38;5;199m0\u001b[0m"]
[67.01579200000005, "o", "\u001b[38;5;199mf\u001b[0m"]
[67.01585900000005, "o", "\u001b[38;5;164mb\u001b[0m"]
[67.01587200000004, "o", "\u001b[38;5;164md\u001b[0m"]
[67.01589400000005, "o", "\u001b[38;5;164m1\u001b[0m"]
[67.01592100000005, "o", "\u001b[38;5;164md\u001b[0m"]
[67.01597900000004, "o", "\u001b[38;5;129m9\u001b[0m"]
[67.01600600000005, "o", "\u001b[38;5;129m2\u001b[0m"]
[67.01603500000004, "o", "\u001b[38;5;129m1\u001b[0m"]
[67.01604700000004, "o", "\u001b[38;5;93m8\u001b[0m"]
[67.01609600000005, "o", "\u001b[38;5;93m1\u001b[0m"]
[67.01615800000005, "o", "\u001b[38;5;93m1\u001b[0m"]
[67.01619200000005, "o", "\u001b[38;5;93m8\u001b[0m"]
[67.01623000000005, "o", "\u001b[38;5;63m1\u001b[0m"]
[67.01626100000006, "o", "\u001b[38;5;63m7\u001b[0m"]
[67.01628500000005, "o", "\u001b[38;5;63md\u001b[0m"]
[67.01630200000005, "o", "\u001b[38;5;63m9\u001b[0m"]
[67.01632700000005, "o", "\u001b[38;5;33m3\u001b[0m"]
[67.01634200000005, "o", "\u001b[38;5;33m5\u001b[0m"]
[67.01637200000005, "o", "\u001b[38;5;33m3\u001b[0m"]
[67.01639500000005, "o", "\u001b[38;5;39m1\u001b[0m"]
[67.01642600000005, "o", "\u001b[38;5;39m0\u001b[0m"]
[67.01645300000006, "o", "\u001b[38;5;39mb\u001b[0m\u001b[38;5;44m3\u001b[0m"]
[67.01647600000005, "o", "\u001b[38;5;44m4\u001b[0m"]
[67.01649800000006, "o", "\u001b[38;5;44m \u001b[0m"]
[67.01651900000006, "o", "\u001b[38;5;44m \u001b[0m"]
[67.01656700000007, "o", "\u001b[38;5;49m/\u001b[0m"]
[67.01662400000006, "o", "\u001b[38;5;49me\u001b[0m"]
[67.01662700000007, "o", "\u001b[38;5;49mt\u001b[0m"]
[67.01666400000008, "o", "\u001b[38;5;48mc\u001b[0m"]
[67.01668900000007, "o", "\u001b[38;5;48m/\u001b[0m"]
[67.01670600000007, "o", "\u001b[38;5;48mf\u001b[0m"]
[67.01672300000007, "o", "\u001b[38;5;84ms\u001b[0m"]
[67.01674400000007, "o", "\u001b[38;5;83mt\u001b[0m"]
[67.01676200000007, "o", "\u001b[38;5;83ma\u001b[0m"]
[67.01680800000007, "o", "\u001b[38;5;83mb\u001b[0m"]
[67.01690500000007, "o", "\u001b[38;5;119m.\u001b[0m"]
[67.01693900000006, "o", "\u001b[38;5;118mh\u001b[0m"]
[67.01700300000006, "o", "\u001b[38;5;118md\u001b[0m\r\r\n"]
[67.01707100000006, "o", "\u001b[38;5;208m4\u001b[0m"]
[67.01709300000006, "o", "\u001b[38;5;208m4\u001b[0m"]
[67.01717400000005, "o", "\u001b[38;5;208m2\u001b[0m"]
[67.01720300000005, "o", "\u001b[38;5;208ma\u001b[0m"]
[67.01721300000005, "o", "\u001b[38;5;203m5\u001b[0m"]
[67.01725500000005, "o", "\u001b[38;5;203mb\u001b[0m"]
[67.01726000000005, "o", "\u001b[38;5;203mc\u001b[0m"]
[67.01729000000005, "o", "\u001b[38;5;203m4\u001b[0m"]
[67.01732300000005, "o", "\u001b[38;5;198m1\u001b[0m"]
[67.01734600000005, "o", "\u001b[38;5;198m7\u001b[0m"]
[67.01739200000004, "o", "\u001b[38;5;198m4\u001b[0m"]
[67.01741700000004, "o", "\u001b[38;5;199ma\u001b[0m"]
[67.01743300000004, "o", "\u001b[38;5;199m8\u001b[0m"]
[67.01746600000004, "o", "\u001b[38;5;199mf\u001b[0m"]
[67.01750100000004, "o", "\u001b[38;5;164m4\u001b[0m"]
[67.01751600000004, "o", "\u001b[38;5;164md\u001b[0m"]
[67.01755100000004, "o", "\u001b[38;5;164m6\u001b[0m"]
[67.01761400000004, "o", "\u001b[38;5;164me\u001b[0m"]
[67.01764900000003, "o", "\u001b[38;5;129mf\u001b[0m"]
[67.01769100000003, "o", "\u001b[38;5;129m8\u001b[0m"]
[67.01769400000003, "o", "\u001b[38;5;129md\u001b[0m"]
[67.01771800000003, "o", "\u001b[38;5;93m5\u001b[0m"]
[67.01774200000003, "o", "\u001b[38;5;93ma\u001b[0m"]
[67.01780800000003, "o", "\u001b[38;5;93me\u001b[0m"]
[67.01783800000003, "o", "\u001b[38;5;93m5\u001b[0m"]
[67.01786200000002, "o", "\u001b[38;5;63md\u001b[0m"]
[67.01788600000002, "o", "\u001b[38;5;63ma\u001b[0m"]
[67.01789400000001, "o", "\u001b[38;5;63m9\u001b[0m"]
[67.01791400000002, "o", "\u001b[38;5;63m2\u001b[0m"]
[67.01793500000002, "o", "\u001b[38;5;33m5\u001b[0m"]
[67.01795700000002, "o", "\u001b[38;5;33m1\u001b[0m"]
[67.01799600000002, "o", "\u001b[38;5;33me\u001b[0m"]
[67.01801000000002, "o", "\u001b[38;5;39mb\u001b[0m"]
[67.01802700000002, "o", "\u001b[38;5;39mb\u001b[0m"]
[67.01805100000001, "o", "\u001b[38;5;39m6\u001b[0m"]
[67.01805500000002, "o", "\u001b[38;5;44ma\u001b[0m"]
[67.01808400000002, "o", "\u001b[38;5;44mb\u001b[0m"]
[67.01810200000001, "o", "\u001b[38;5;44m4\u001b[0m"]
[67.01813200000001, "o", "\u001b[38;5;44m5\u001b[0m"]
[67.01814, "o", "\u001b[38;5;49m5\u001b[0m"]
[67.018157, "o", "\u001b[38;5;49m \u001b[0m"]
[67.018169, "o", "\u001b[38;5;49m \u001b[0m"]
[67.018187, "o", "\u001b[38;5;48m/\u001b[0m"]
[67.01822, "o", "\u001b[38;5;48me\u001b[0m"]
[67.018248, "o", "\u001b[38;5;48mt\u001b[0m"]
[67.01826, "o", "\u001b[38;5;84mc\u001b[0m"]
[67.018283, "o", "\u001b[38;5;83m/\u001b[0m"]
[67.018316, "o", "\u001b[38;5;83mf\u001b[0m\u001b[38;5;83mt\u001b[0m"]
[67.018338, "o", "\u001b[38;5;119mp\u001b[0m"]
[67.018354, "o", "\u001b[38;5;118md\u001b[0m"]
[67.018371, "o", "\u001b[38;5;118m.\u001b[0m"]
[67.018385, "o", "\u001b[38;5;118mc\u001b[0m"]
[67.01839799999999, "o", "\u001b[38;5;154mo\u001b[0m"]
[67.018419, "o", "\u001b[38;5;154mn\u001b[0m"]
[67.018446, "o", "\u001b[38;5;154mf\u001b[0m\r\r\n"]
[67.018504, "o", "\u001b[38;5;208m4\u001b[0m"]
[67.018509, "o", "\u001b[38;5;203m4\u001b[0m"]
[67.018519, "o", "\u001b[38;5;203m2\u001b[0m"]
[67.018575, "o", "\u001b[38;5;203ma\u001b[0m"]
[67.018578, "o", "\u001b[38;5;203m5\u001b[0m"]
[67.018591, "o", "\u001b[38;5;198mb\u001b[0m"]
[67.018601, "o", "\u001b[38;5;198mc\u001b[0m"]
[67.018613, "o", "\u001b[38;5;198m4\u001b[0m"]
[67.018648, "o", "\u001b[38;5;199m1\u001b[0m"]
[67.018671, "o", "\u001b[38;5;199m7\u001b[0m"]
[67.018675, "o", "\u001b[38;5;199m4\u001b[0m"]
[67.018709, "o", "\u001b[38;5;164ma\u001b[0m"]
[67.01871200000001, "o", "\u001b[38;5;164m8\u001b[0m"]
[67.01873300000001, "o", "\u001b[38;5;164mf\u001b[0m"]
[67.01874500000001, "o", "\u001b[38;5;164m4\u001b[0m"]
[67.01878500000001, "o", "\u001b[38;5;129md\u001b[0m"]
[67.01879600000001, "o", "\u001b[38;5;129m6\u001b[0m"]
[67.01883000000001, "o", "\u001b[38;5;129me\u001b[0m"]
[67.01883900000001, "o", "\u001b[38;5;93mf\u001b[0m"]
[67.01885400000002, "o", "\u001b[38;5;93m8\u001b[0m"]
[67.01887000000002, "o", "\u001b[38;5;93md\u001b[0m"]
[67.01890000000002, "o", "\u001b[38;5;93m5\u001b[0m"]
[67.01892600000002, "o", "\u001b[38;5;63ma\u001b[0m"]
[67.01898200000002, "o", "\u001b[38;5;63me\u001b[0m"]
[67.01899800000002, "o", "\u001b[38;5;63m5\u001b[0m"]
[67.01903000000003, "o", "\u001b[38;5;63md\u001b[0m"]
[67.01905300000003, "o", "\u001b[38;5;33ma\u001b[0m"]
[67.01907400000003, "o", "\u001b[38;5;33m9\u001b[0m"]
[67.01909300000003, "o", "\u001b[38;5;33m2\u001b[0m"]
[67.01911800000002, "o", "\u001b[38;5;39m5\u001b[0m"]
[67.01913900000002, "o", "\u001b[38;5;39m1\u001b[0m"]
[67.01915900000003, "o", "\u001b[38;5;39me\u001b[0m"]
[67.01918700000003, "o", "\u001b[38;5;44mb\u001b[0m"]
[67.01922700000003, "o", "\u001b[38;5;44mb\u001b[0m"]
[67.01924000000002, "o", "\u001b[38;5;44m6\u001b[0m"]
[67.01925900000002, "o", "\u001b[38;5;44ma\u001b[0m"]
[67.01929100000002, "o", "\u001b[38;5;49mb\u001b[0m"]
[67.01931200000003, "o", "\u001b[38;5;49m4\u001b[0m"]
[67.01933000000002, "o", "\u001b[38;5;49m5\u001b[0m"]
[67.01934500000003, "o", "\u001b[38;5;48m5\u001b[0m"]
[67.01935900000002, "o", "\u001b[38;5;48m \u001b[0m"]
[67.01937700000002, "o", "\u001b[38;5;48m \u001b[0m"]
[67.01939300000002, "o", "\u001b[38;5;84m/\u001b[0m"]
[67.01941100000002, "o", "\u001b[38;5;83me\u001b[0m"]
[67.01944200000003, "o", "\u001b[38;5;83mt\u001b[0m"]
[67.01946100000002, "o", "\u001b[38;5;83mc\u001b[0m"]
[67.01948400000002, "o", "\u001b[38;5;119m/\u001b[0m"]
[67.01950200000002, "o", "\u001b[38;5;118mf\u001b[0m"]
[67.01952200000002, "o", "\u001b[38;5;118mt\u001b[0m"]
[67.01953700000003, "o", "\u001b[38;5;118mp\u001b[0m"]
[67.01956800000004, "o", "\u001b[38;5;154md\u001b[0m"]
[67.01958600000003, "o", "\u001b[38;5;154m.\u001b[0m"]
[67.01963500000004, "o", "\u001b[38;5;154mc\u001b[0m\u001b[38;5;184mo\u001b[0m"]
[67.01965300000003, "o", "\u001b[38;5;184mn\u001b[0m"]
[67.01967400000004, "o", "\u001b[38;5;184mf\u001b[0m"]
[67.01969500000004, "o", "\u001b[38;5;184m.\u001b[0m"]
[67.01970700000004, "o", "\u001b[38;5;214md\u001b[0m"]
[67.01972600000003, "o", "\u001b[38;5;214me\u001b[0m"]
[67.01975100000003, "o", "\u001b[38;5;214mf\u001b[0m"]
[67.01983100000002, "o", "\u001b[38;5;208ma\u001b[0m"]
[67.01985500000002, "o", "\u001b[38;5;208mu\u001b[0m"]
[67.01986700000002, "o", "\u001b[38;5;208ml\u001b[0m"]
[67.01990200000002, "o", "\u001b[38;5;209mt\u001b[0m\r\r\n"]
[67.01993900000002, "o", "\u001b[38;5;203md\u001b[0m"]
[67.01996900000002, "o", "\u001b[38;5;203m3\u001b[0m"]
[67.01998000000002, "o", "\u001b[38;5;198me\u001b[0m"]
[67.02001100000003, "o", "\u001b[38;5;198m5\u001b[0m"]
[67.02005400000003, "o", "\u001b[38;5;198mf\u001b[0m"]
[67.02009500000003, "o", "\u001b[38;5;199mb\u001b[0m"]
[67.02012000000002, "o", "\u001b[38;5;199m0\u001b[0m"]
[67.02014000000003, "o", "\u001b[38;5;199mc\u001b[0m"]
[67.02017400000003, "o", "\u001b[38;5;164m5\u001b[0m"]
[67.02018800000002, "o", "\u001b[38;5;164m8\u001b[0m"]
[67.02022000000002, "o", "\u001b[38;5;164m2\u001b[0m"]
[67.02024600000003, "o", "\u001b[38;5;164m6\u001b[0m"]
[67.02028900000003, "o", "\u001b[38;5;129m4\u001b[0m"]
[67.02030500000004, "o", "\u001b[38;5;129m5\u001b[0m"]
[67.02034600000003, "o", "\u001b[38;5;129me\u001b[0m"]
[67.02037000000003, "o", "\u001b[38;5;93m6\u001b[0m"]
[67.02038000000003, "o", "\u001b[38;5;93m0\u001b[0m"]
[67.02040000000004, "o", "\u001b[38;5;93mf\u001b[0m"]
[67.02042300000004, "o", "\u001b[38;5;93m8\u001b[0m"]
[67.02044100000003, "o", "\u001b[38;5;63ma\u001b[0m"]
[67.02048100000003, "o", "\u001b[38;5;63m1\u001b[0m"]
[67.02051500000003, "o", "\u001b[38;5;63m3\u001b[0m"]
[67.02051900000004, "o", "\u001b[38;5;63m8\u001b[0m"]
[67.02054100000004, "o", "\u001b[38;5;33m0\u001b[0m"]
[67.02055900000003, "o", "\u001b[38;5;33m2\u001b[0m"]
[67.02057800000003, "o", "\u001b[38;5;33mb\u001b[0m"]
[67.02060200000003, "o", "\u001b[38;5;39me\u001b[0m"]
[67.02062700000002, "o", "\u001b[38;5;39m0\u001b[0m"]
[67.02065300000002, "o", "\u001b[38;5;39mc\u001b[0m"]
[67.02067400000003, "o", "\u001b[38;5;44m9\u001b[0m"]
[67.02069400000003, "o", "\u001b[38;5;44m0\u001b[0m"]
[67.02071500000004, "o", "\u001b[38;5;44m9\u001b[0m"]
[67.02103500000004, "o", "\u001b[38;5;44ma\u001b[0m\u001b[38;5;49m3\u001b[0m"]
[67.02105000000005, "o", "\u001b[38;5;49mf\u001b[0m"]
[67.02108000000004, "o", "\u001b[38;5;49m9\u001b[0m"]
[67.02109500000005, "o", "\u001b[38;5;48me\u001b[0m"]
[67.02110400000005, "o", "\u001b[38;5;48m4\u001b[0m"]
[67.02113100000005, "o", "\u001b[38;5;48md\u001b[0m"]
[67.02114300000005, "o", "\u001b[38;5;84m7\u001b[0m"]
[67.02117400000006, "o", "\u001b[38;5;83m \u001b[0m"]
[67.02118900000006, "o", "\u001b[38;5;83m \u001b[0m"]
[67.02122900000006, "o", "\u001b[38;5;83m/\u001b[0m"]
[67.02126300000006, "o", "\u001b[38;5;119me\u001b[0m"]
[67.02127600000006, "o", "\u001b[38;5;118mt\u001b[0m"]
[67.02128800000006, "o", "\u001b[38;5;118mc\u001b[0m"]
[67.02131000000006, "o", "\u001b[38;5;118m/\u001b[0m"]
[67.02135900000006, "o", "\u001b[38;5;154mf\u001b[0m"]
[67.02138000000006, "o", "\u001b[38;5;154mt\u001b[0m\u001b[38;5;154mp\u001b[0m"]
[67.02140200000007, "o", "\u001b[38;5;184mu\u001b[0m"]
[67.02142500000006, "o", "\u001b[38;5;184ms\u001b[0m"]
[67.02144700000007, "o", "\u001b[38;5;184me\u001b[0m"]
[67.02146400000007, "o", "\u001b[38;5;184mr\u001b[0m"]
[67.02147900000007, "o", "\u001b[38;5;214ms\u001b[0m\r\r\n"]
[67.02441700000007, "o", "bash-3.2$ "]
[69.02441700000007, "o", "#"]
[69.32766700000008, "o", " "]
[69.83990200000008, "o", "T"]
[70.16800600000008, "o", "o"]
[70.31195300000007, "o", " "]
[70.49585400000008, "o", "f"]
[70.55982100000008, "o", "i"]
[70.71184300000009, "o", "n"]
[70.77620200000008, "o", "i"]
[70.87204300000008, "o", "s"]
[71.00817700000007, "o", "h"]
[71.06370700000008, "o", " "]
[71.22400800000008, "o", "r"]
[71.27981700000008, "o", "e"]
[71.46387800000008, "o", "c"]
[71.59178900000008, "o", "o"]
[71.71193000000008, "o", "r"]
[71.91982000000009, "o", "d"]
[72.09602300000009, "o", "i"]
[72.17625000000008, "o", "n"]
[72.33599900000009, "o", "g"]
[72.47186000000009, "o", " "]
[72.97607100000009, "o", "j"]
[73.15967000000009, "o", "u"]
[73.26397500000009, "o", "s"]
[73.39978400000008, "o", "t"]
[73.47253900000008, "o", " "]
[73.59968900000008, "o", "e"]
[73.79956100000008, "o", "x"]
[73.92801700000008, "o", "i"]
[74.13601000000008, "o", "t"]
[74.64808800000009, "o", " "]
[75.13573700000009, "o", "t"]
[75.25604900000009, "o", "h"]
[75.31966500000009, "o", "e"]
[75.40762900000009, "o", " "]
[75.56767100000009, "o", "s"]
[75.67958500000009, "o", "h"]
[75.81597000000009, "o", "e"]
[75.93592900000009, "o", "l"]
[76.11718000000009, "o", "l"]
[76.99980200000009, "o", "\r\r\n"]
[76.9999010000001, "o", "bash-3.2$ "]
[77.74379100000009, "o", "e"]
[77.94356500000009, "o", "x"]
[78.03988400000009, "o", "i"]
[78.28786700000009, "o", "t"]
[79.87125700000009, "o", "\r\r\n"]
[79.87133300000009, "o", "exit\r\r\n"]
[79.87343400000009, "o", "\u001b[0;32masciinema: recording finished\u001b[0m\r\n"]
[79.8766910000001, "o", "\u001b[0;32masciinema: press <enter> to upload to asciinema.org, <ctrl-c> to save locally\u001b[0m\r\n"]
[81.8766910000001, "o", "\r\n"]
[83.3828910000001, "o", "https://asciinema.org/a/17648\r\n"]
[83.3847300000001, "o", "> "]
[85.3847300000001, "o", "#"]
[85.5058490000001, "o", " "]
[86.0244440000001, "o", "O"]
[86.3288060000001, "o", "p"]
[86.4807220000001, "o", "e"]
[86.6568070000001, "o", "n"]
[86.7754540000001, "o", " "]
[86.9045620000001, "o", "t"]
[87.0967350000001, "o", "h"]
[87.15255000000009, "o", "e"]
[87.24893000000009, "o", " "]
[87.41678500000009, "o", "a"]
[87.51268400000009, "o", "b"]
[87.7125420000001, "o", "o"]
[87.8326160000001, "o", "v"]
[87.8748970000001, "o", "e"]
[87.9609480000001, "o", " "]
[88.4808400000001, "o", "U"]
[88.5767220000001, "o", "R"]
[88.7287400000001, "o", "L"]
[88.9609520000001, "o", " "]
[89.08073300000011, "o", "t"]
[89.19243700000011, "o", "o"]
[89.28091700000012, "o", " "]
[89.42460600000011, "o", "v"]
[89.64845300000012, "o", "i"]
[89.72877900000012, "o", "e"]
[89.81626300000012, "o", "w"]
[89.86428900000011, "o", " "]
[89.98418500000011, "o", "t"]
[90.0962860000001, "o", "h"]
[90.18420900000011, "o", "e"]
[90.62416100000011, "o", " "]
[90.83203000000012, "o", "r"]
[90.89643500000011, "o", "e"]
[91.08026600000011, "o", "c"]
[91.2642560000001, "o", "o"]
[91.37675800000011, "o", "r"]
[91.5527270000001, "o", "d"]
[91.6566550000001, "o", "i"]
[91.7282520000001, "o", "n"]
[91.8165970000001, "o", "g"]
[92.7527270000001, "o", "\r\n"]
[92.7528120000001, "o", "> "]
[94.7528120000001, "o", "#"]
[94.82492400000011, "o", " "]
[95.06485500000011, "o", "N"]
[95.35267900000011, "o", "o"]
[95.4489120000001, "o", "w"]
[95.78491500000011, "o", " "]
[95.96899400000011, "o", "i"]
[96.03293900000011, "o", "n"]
[96.06506600000012, "o", "s"]
[96.27268500000011, "o", "t"]
[96.44066900000011, "o", "a"]
[96.56056700000012, "o", "l"]
[96.71447000000012, "o", "l"]
[96.79361700000013, "o", " "]
[96.99243700000012, "o", "a"]
[97.06484700000013, "o", "s"]
[97.14443700000012, "o", "c"]
[97.32889200000012, "o", "i"]
[97.47249500000012, "o", "i"]
[97.94490400000012, "o", "n"]
[98.39262200000012, "o", "e"]
[98.60046200000012, "o", "m"]
[98.71262100000013, "o", "a"]
[98.88086700000012, "o", " "]
[99.33675600000012, "o", "a"]
[99.48085300000012, "o", "n"]
[99.57670300000012, "o", "d"]
[99.66481600000013, "o", " "]
[99.80873400000013, "o", "s"]
[99.92880100000013, "o", "t"]
[100.01647300000013, "o", "a"]
[100.10446400000014, "o", "r"]
[100.28048500000014, "o", "t"]
[100.36057400000014, "o", " "]
[100.49670100000014, "o", "r"]
[100.56069700000015, "o", "e"]
[100.72863300000014, "o", "c"]
[100.87216700000015, "o", "o"]
[100.95221800000014, "o", "r"]
[101.12867800000015, "o", "d"]
[101.25627100000015, "o", "i"]
[101.33647000000015, "o", "n"]
[101.38467800000015, "o", "g"]
[101.48070800000015, "o", " "]
[101.64064600000015, "o", "y"]
[101.84869600000015, "o", "o"]
[101.89790400000014, "o", "u"]
[101.97655300000014, "o", "r"]
[102.04865900000014, "o", " "]
[102.22439700000014, "o", "o"]
[102.32021300000014, "o", "w"]
[102.54436600000014, "o", "n"]
[102.60843400000014, "o", " "]
[102.71200100000014, "o", "s"]
[102.88040500000014, "o", "e"]
[103.07197100000013, "o", "s"]
[103.20026200000014, "o", "s"]
[103.36005000000014, "o", "i"]
[103.39190300000014, "o", "o"]
[103.86409800000014, "o", "n"]
[104.14407800000014, "o", "s"]
[105.20833500000013, "o", "\r\n"]
[105.20846100000013, "o", "> "]
[107.20846100000013, "o", "#"]
[107.31221900000013, "o", " "]
[107.53632600000013, "o", "O"]
[107.83240800000013, "o", "h"]
[108.07205200000013, "o", ","]
[108.20041500000012, "o", " "]
[108.35230800000012, "o", "a"]
[108.47197400000012, "o", "n"]
[108.53644000000011, "o", "d"]
[108.61650800000011, "o", " "]
[108.81636000000012, "o", "y"]
[109.01618200000011, "o", "o"]
[109.06742300000012, "o", "u"]
[109.16039300000011, "o", " "]
[109.32832200000011, "o", "c"]
[109.37630400000012, "o", "a"]
[109.52022800000012, "o", "n"]
[109.64020200000012, "o", " "]
[109.87184300000011, "o", "c"]
[110.06422900000011, "o", "o"]
[110.14403000000011, "o", "p"]
[111.03329100000012, "o", "y"]
[111.25009000000011, "o", "-"]
[111.43992500000012, "o", "p"]
[111.55226800000011, "o", "a"]
[111.5840330000001, "o", "s"]
[111.8242570000001, "o", "t"]
[111.9200160000001, "o", "e"]
[112.1041780000001, "o", " "]
[112.5842070000001, "o", "f"]
[112.6388500000001, "o", "r"]
[112.7839900000001, "o", "o"]
[112.8481830000001, "o", "m"]
[112.92026100000011, "o", " "]
[113.1282110000001, "o", "h"]
[113.1696950000001, "o", "e"]
[113.4080690000001, "o", "r"]
[113.4717430000001, "o", "e"]
[115.4717430000001, "o", "\r\n"]
[115.4718100000001, "o", "> "]
[116.0294910000001, "o", "#"]
[116.10441100000011, "o", " "]
[116.47183000000011, "o", "B"]
[116.7599770000001, "o", "y"]
[116.8479770000001, "o", "e"]
[117.5680820000001, "o", "!"]
[119.2079330000001, "o", "\r\n"]
`

func readInputContent() ([]byte, error) {
	_lines := strings.Split(data, "\n")

	var inputs []byte
	for i, line := range _lines {
		if i == 0 {
			continue
		}
		if line == "" {
			continue
		}
		var arr []interface{}
		if err := json.Unmarshal([]byte(line), &arr); err != nil {
			return nil, err
		}
		inputs = append(inputs, []byte(arr[2].(string))...)
	}
	return inputs, nil
}

func main() {
	content, err := readInputContent()
	if err != nil {
		log.Fatal(err)
	}

	v := vt.NewWithOpts(vt.Opts{
		Logger: log.Default(),
	})
	_, err = v.Advance(content)
	if err != nil {
		log.Fatal(err)
	}
	v.Parse()
	lines := v.Result()
	for _, line := range lines {
		println(line)
	}
	v.Reset()
}
