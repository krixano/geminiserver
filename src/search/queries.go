package search

import (
    // "database/sql"
    // "strings"
)

type SearchQueryOptions struct {
    limit int // 0 = no limit and no pages
    page int
}

/*
func matchExpressions(searchWords []string, table string, id_col string) func(row string) string {
    return func(row string) string {
        var builder strings.Builder
        for i := 0; i < len(searchWords); i++ {
            word := strings.ReplaceAll(strings.ReplaceAll(searchWords[i], "'", "''"), "%", "[%]")
            negate := false
            if strings.HasPrefix(word, "-") {
                negate = true
                word = word[1:]
            }

            fmt.Fprintf(&builder, "(SELECT COUNT(" + row + ") FROM " + table + " AS " + row + "match" + i + " WHERE " + row + (negate ? " NOT" : "") + " LIKE '%" + word + "%' AND " + table + "." + row + "=" + row + "match" + i + "." + row + " AND " + table + "." + id_col + "=" + row + "match" + i + "." + id_col + " AND " + table + ".json_id=" + row + "match" + i + ".json_id) AS " + row + "match" + i)

            //"(SELECT COUNT(" + row + ") FROM " + table + " AS " + row + "match" + i + joinJson + " WHERE " + (usingJson ? "json_" + row + "match" + i + "." + row : row) + (negate ? " NOT" : "") + " LIKE '%" + word + "%' AND " + (usingJson ? json_name : table) + "." + row + "=" + (usingJson ? "json_" : "") + row + "match" + i + "." + row + " AND " + table + "." + id_col + "=" + row + "match" + i + "." + id_col + " AND " + table + ".json_id=" + row + "match" + i + ".json_id) AS " + row + "match" + i;
            if i != len(searchWords) - 1 {
                fmt.Fprintf(&builder, ", ")
            }
        }

        return builder.String()
    }*/
    /*var joinJson = "";
    if (usingJson) {
        joinJson = " LEFT JOIN json AS json_" + row + "match" + i + " USING (json_id) "
    }*/
//}


/*
func matches(searchWords []string) func(row string, multiplication int) string {
    return func(row string, multiplication int) string {
        var builder strings.Builder
        for i := 0; i < len(searchWords); i++ {
            if !multiplication {
                fmt.Fprintf(&builder, row + "match" + i)
            } else {
                fmt.Fprintf(&builder, "(" + row + "match" + i + " * " + multiplication + ")")
            }
            if i != len(searchWords) - 1 {
                fmt.Fprintf(&builder, " + ")
            }
        }

        return builder.String()
    }
}
*/

/*
func searchDbQuery(conn *sql.DB, searchQuery string, options SearchQueryOptions) string {
    var offset = 0;
    if (options.limit && options.limit != 0)
        offset = (options.page || 0) * options.limit;
    var searchWords = searchQuery.split(" ").map((s) => {
        return s.replace(/'/g, "''").replace(/%/g, "[%]");//.toLowerCase();
    }).filter((s) => {
        return s != "";
    });

    if (searchWords.length == 0) searchWords = [""];

    var matchExpressions_inner = matchExpressions(searchWords, options.table, options.id_col);
    var matches_inner = matches(searchWords);

    var searchSelects = "";
    var searchMatchesAdded = "";
    var searchMatchesOrderBy = "";
    for (var i = 0; i < options.searchSelects.length; i++) {
        var col = options.searchSelects[i].col;
        var score = options.searchSelects[i].score;
        var select = options.searchSelects[i].select;
        var matchName = options.searchSelects[i].matchName;
        var inSearchMatchesAdded = options.searchSelects[i].inSearchMatchesAdded != undefined ? options.searchSelects[i].inSearchMatchesAdded : true;
        var inSearchMatchesOrderBy = options.searchSelects[i].inSearchMatchesOrderBy != undefined ? options.searchSelects[i].inSearchMatchesOrderBy : true;
        var skip = options.searchSelects[i].skip != undefined ? options.searchSelects[i].skip : false;
        var usingJson = options.searchSelects[i].usingJson != undefined ? options.searchSelects[i].usingJson : false;
        var having = options.searchSelects[i].having;

        if (!skip) {
            if (i != 0) {
                searchSelects += ", ";
            }
            if (select) {
                searchSelects += "(" + select + ") AS " + col;
            } else {
                searchSelects += matchExpressions_inner(col, usingJson);
            }
        }

        if (inSearchMatchesAdded && !skip) {
            if (i != 0) {
                searchMatchesAdded += " + ";
            }
            if (select) {
                searchMatchesAdded += col;
            } else {
                searchMatchesAdded += matches_inner(col);
            }
        }

        if (options.orderByScore) {
            if (searchMatchesOrderBy == "")
                searchMatchesOrderBy = "(";
            if (inSearchMatchesOrderBy && !skip) {
                if (i != 0) {
                    searchMatchesOrderBy += " + ";
                }
                if (select) {
                    searchMatchesOrderBy += "(" + col + " * " + score + ")";
                } else {
                    searchMatchesOrderBy += matches_inner(col, score);
                }
            }
        }
    }

    searchMatchesOrderBy += ") " + (options.orderDirection || "DESC");

    var beforeOrderBy = "";
    var afterOrderBy = "";
    if (options.beforeOrderBy) {
        beforeOrderBy = options.beforeOrderBy + ", ";
    }
    if (options.afterOrderBy) {
        if (options.orderByScore && searchMatchesOrderBy) {
            afterOrderBy = ", " + options.afterOrderBy;
        } else {
            afterOrderBy = "ORDER BY " + options.afterOrderBy;
        }
    }

    var builder strings.Builder

    var query = `
        SELECT ${options.select || "*"}
            ${searchSelects ? ", " + searchSelects : ""}
        FROM ${options.table}
        ${options.join || ""}
        WHERE ${options.where || ""}
            ${options.where && searchMatchesAdded ? "AND" : ""}
            ${searchMatchesAdded ? "(" + searchMatchesAdded + ") > 0" : ""}
        ${options.groupBy ? "GROUP BY " + options.groupBy : ""}
        ${options.having ? "HAVING " + options.having : ""}
        ${options.orderByScore && searchMatchesOrderBy ? "ORDER BY " + beforeOrderBy + searchMatchesOrderBy : ""} ${afterOrderBy}
        ${options.limit ? "LIMIT " + options.limit : ""}
        ${options.limit ? "OFFSET " + offset : ""}
        `;
    console.log(query);
    return query;
}

function matchExpressions(searchWords, table, id_col) {
    return (row, usingJson = false) => {
        var expressions = "";
        for (var i = 0; i < searchWords.length; i++) {
            var word = searchWords[i].replace(/'/g, "''").replace(/%/g, "[%]");
            var negate = false;
            if (word[0] == "-") {
                negate = true;
                word = word.slice(1);
                console.log(word);
            }
            var joinJson = "";
            if (usingJson) {
                joinJson = " LEFT JOIN json AS json_" + row + "match" + i + " USING (json_id) "
            }
            var json_name = "json";
            if (typeof usingJson === "string") json_name = usingJson;
            expressions += "(SELECT COUNT(" + row + ") FROM " + table + " AS " + row + "match" + i + joinJson + " WHERE " + (usingJson ? "json_" + row + "match" + i + "." + row : row) + (negate ? " NOT" : "") + " LIKE '%" + word + "%' AND " + (usingJson ? json_name : table) + "." + row + "=" + (usingJson ? "json_" : "") + row + "match" + i + "." + row + " AND " + table + "." + id_col + "=" + row + "match" + i + "." + id_col + " AND " + table + ".json_id=" + row + "match" + i + ".json_id) AS " + row + "match" + i;
            if (i != searchWords.length - 1) {
                expressions += ", ";
            }
        }
        return expressions;
    }
}

function matches(searchWords) {
    return (row, multiplication) => {
        var s = "";
        for (var i = 0; i < searchWords.length; i++) {
            if (!multiplication) {
                s += row + "match" + i;
            } else {
                s += "(" + row + "match" + i + " * " + multiplication + ")";
            }
            if (i != searchWords.length - 1) {
                s += " + ";
            }
        }
        return s;
    }
}

// TODO: Rename searchMatchesAdded and searchMatchesOrderBy
// limit = 0, no limit
function searchDbQuery(zeroframe, searchQuery, options) {
    var offset = 0;
    if (options.limit && options.limit != 0)
        offset = (options.page || 0) * options.limit;
    var searchWords = searchQuery.split(" ").map((s) => {
        return s.replace(/'/g, "''").replace(/%/g, "[%]");//.toLowerCase();
    }).filter((s) => {
        return s != "";
    });

    if (searchWords.length == 0) searchWords = [""];

    var matchExpressions_inner = matchExpressions(searchWords, options.table, options.id_col);
    var matches_inner = matches(searchWords);

    var searchSelects = "";
    var searchMatchesAdded = "";
    var searchMatchesOrderBy = "";
    for (var i = 0; i < options.searchSelects.length; i++) {
        var col = options.searchSelects[i].col;
        var score = options.searchSelects[i].score;
        var select = options.searchSelects[i].select;
        var matchName = options.searchSelects[i].matchName;
        var inSearchMatchesAdded = options.searchSelects[i].inSearchMatchesAdded != undefined ? options.searchSelects[i].inSearchMatchesAdded : true;
        var inSearchMatchesOrderBy = options.searchSelects[i].inSearchMatchesOrderBy != undefined ? options.searchSelects[i].inSearchMatchesOrderBy : true;
        var skip = options.searchSelects[i].skip != undefined ? options.searchSelects[i].skip : false;
        var usingJson = options.searchSelects[i].usingJson != undefined ? options.searchSelects[i].usingJson : false;
        var having = options.searchSelects[i].having;

        if (!skip) {
            if (i != 0) {
                searchSelects += ", ";
            }
            if (select) {
                searchSelects += "(" + select + ") AS " + col;
            } else {
                searchSelects += matchExpressions_inner(col, usingJson);
            }
        }

        if (inSearchMatchesAdded && !skip) {
            if (i != 0) {
                searchMatchesAdded += " + ";
            }
            if (select) {
                searchMatchesAdded += col;
            } else {
                searchMatchesAdded += matches_inner(col);
            }
        }

        if (options.orderByScore) {
            if (searchMatchesOrderBy == "")
                searchMatchesOrderBy = "(";
            if (inSearchMatchesOrderBy && !skip) {
                if (i != 0) {
                    searchMatchesOrderBy += " + ";
                }
                if (select) {
                    searchMatchesOrderBy += "(" + col + " * " + score + ")";
                } else {
                    searchMatchesOrderBy += matches_inner(col, score);
                }
            }
        }
    }

    searchMatchesOrderBy += ") " + (options.orderDirection || "DESC");

    var beforeOrderBy = "";
    var afterOrderBy = "";
    if (options.beforeOrderBy) {
        beforeOrderBy = options.beforeOrderBy + ", ";
    }
    if (options.afterOrderBy) {
        if (options.orderByScore && searchMatchesOrderBy) {
            afterOrderBy = ", " + options.afterOrderBy;
        } else {
            afterOrderBy = "ORDER BY " + options.afterOrderBy;
        }
    }

    var query = `
        SELECT ${options.select || "*"}
            ${searchSelects ? ", " + searchSelects : ""}
        FROM ${options.table}
        ${options.join || ""}
        WHERE ${options.where || ""}
            ${options.where && searchMatchesAdded ? "AND" : ""}
            ${searchMatchesAdded ? "(" + searchMatchesAdded + ") > 0" : ""}
        ${options.groupBy ? "GROUP BY " + options.groupBy : ""}
        ${options.having ? "HAVING " + options.having : ""}
        ${options.orderByScore && searchMatchesOrderBy ? "ORDER BY " + beforeOrderBy + searchMatchesOrderBy : ""} ${afterOrderBy}
        ${options.limit ? "LIMIT " + options.limit : ""}
        ${options.limit ? "OFFSET " + offset : ""}
        `;
    console.log(query);
    return query;
}
*/
