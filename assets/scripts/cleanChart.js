function lineChart() {

  //set defaults
  var margin = {
    top: 20,
    right: 20,
    bottom: 20,
    left: 40
  };
  var width = 1000;
  var height = 500;
  var xValue = function (d) {
    return d[0];
  };
  var yValue = function (d) {
    return d[1];
  };

  var xScale = d3.time.scale();
  var yScale = d3.scale.linear();
  var xAxis = d3.svg.axis().scale(xScale).orient("bottom").tickSize(6, 0);
  var yAxis = d3.svg.axis().scale(yScale).orient("left").ticks(20);
  var area = d3.svg.area().x(X).y1(Y).interpolate("step-after").defined(function (d) {
    return !isNaN(d[1]);
  });
  var line = d3.svg.line().interpolate("step-after").x(X).y(Y).defined(function (d) {
    return !isNaN(d[1]);
  });
  var zoom = d3.behavior.zoom().scaleExtent([1, 10]).on("zoom", zoomed);

  //var zoom = d3.behavior.zoom().x(xScale).y(yScale).on("zoom", zoomed);
  var drawLine = true; //line or no line?
  var drawArea = true; //area graph or no?
  var drawPoints = true; //points or no?
  var drawXAxis = true;
  var drawYAxis = true;

  var zoomTrans = [0, 0];
  var zoomScale = 1;

  /***********************************************************


   ***********************************************************/
  function chart(selection) {

    selection.each(function (data) {

      // Convert data to standard representation greedily;
      // this is needed for nondeterministic accessors.
      data = data.map(function (d, i) {
        return [xValue.call(data, d, i), yValue.call(data, d, i)];
      });

      //update the X&Y scales to match the data range
      updateXScale(data);
      updateYScale(data);

      // Select the svg element, if it exists.
      var svg = d3.select(this)
        .selectAll("svg")
        .data([data]);

      //update zoom scale values
      zoom.x(xScale)
        .y(yScale)
      //  .call(xExtent(-200, 200))
      //  .yExtent([-150, 150])
        .on("zoom", zoomed);

      // Otherwise, create the skeletal chart.
      var gEnter = svg.enter()
        .append("svg")
        .attr("width", '100%')
        .attr("height", '100%')
        .attr('viewBox', '0 0 ' + Math.min(width, height) + ' ' + Math.min(width, height))
        .attr('preserveAspectRatio', 'xMinYMin')
        .append("g")
        .attr("transform", "translate(" + Math.min(width, height) / 2 + "," + Math.min(width, height) / 2 + ")");;

      //add clipPath so line/area don't overflow outside chart
      gEnter.append("clipPath")
        .attr("id", "clip")
        .append("rect")
        .attr("x", 0)
        .attr("y", 0)
        .attr("width", width - margin.left - margin.right)
        .attr("height", height - margin.top - margin.bottom);

      //add line/area and axis
      if (drawArea) gEnter.append("path").attr("class", "area").attr("clip-path", "url(#clip)");
      if (drawLine) gEnter.append("path").attr("class", "line").attr("clip-path", "url(#clip)");
      if (drawXAxis) gEnter.append("g").attr("class", "x axis");
      if (drawYAxis) gEnter.append("g").attr("class", "y axis");

      //chart background
      gEnter.append("rect")
        .attr('class', 'pane')
        .attr("width", width - margin.left - margin.right)
        .attr("height", height - margin.top - margin.bottom);
      //  .attr("fill", "white");

      // Update the outer dimensions.
      svg.attr("width", width)
        .attr("height", height);

      var g = svg.select("g");
      draw(g);

    });
  }

  /***********************************************************

  Draw

   ***********************************************************/
  function draw(g) {

    // Update the inner dimensions.
    g.attr("transform", "translate(" + margin.left + "," + margin.top + ")")
      .call(zoom);

    // Update the area path.
    g.select(".area")
      .attr("d", area.y0(yScale.range()[0]));

    // Update the line path.
    g.select(".line")
      .attr("d", line);

    // Update the x-axis.
    g.select(".x.axis")
      .attr("transform", "translate(0," + yScale.range()[0] + ")")
      .call(xAxis);

    // Update the y-axis.
    g.select(".y.axis")
      .call(yAxis);

  }
  /***********************************************************


   ***********************************************************/
  function updateXScale(dataSet) {

    // Update the x-scale.
    xScale
      .domain(d3.extent(dataSet, function (d) {
        return d[0];
      }))
      .range([0, width - margin.left - margin.right]);

  }
  /***********************************************************


   ***********************************************************/
  function updateYScale(dataSet) {

    // Update the y-scale.
    yScale
      .domain([0, d3.max(dataSet, function (d) {
        return d[1];
      })])
      .range([height - margin.top - margin.bottom, 0]);
  }

  /***********************************************************
   The x-accessor for the path generator; xScale x xValue.

   ***********************************************************/
  function X(d) {
    return xScale(d[0]);
  }


  /***********************************************************
   The y-accessor for the path generator; yScale x yValue.

   ***********************************************************/
  function Y(d) {
    return yScale(d[1]);
  }

  function xExtent(xmin, xmax) {
    if (xScale.domain()[0] < xmin) {
      zoom.translate([zoom.translate()[0] - xScale(xmin) + xScale.range()[0], zoom.translate()[1]]);
    } else if (xScale.domain()[1] > xmax) {
      zoom.translate([zoom.translate()[0] - xScale(xmax) + xScale.range()[1], zoom.translate()[1]]);
    }
  }


  /***********************************************************


   ***********************************************************/
  function zoomed() {
    //console.log("translate: " + d3.event.translate);
    //console.log("scale: " + d3.event.scale);
    //d3.event.transform(xScale);

    zoomScale = d3.event.scale;
    zoomTrans = d3.event.translate;

    //xExtent(-200, 200);

    //zoomTrans[0] = Math.min(width / 2 * (zoomScale - 1), Math.max(width / 2 * (1 - zoomScale), zoomTrans[0]));
    //zoomTrans[1] = Math.min(height / 2 * (zoomScale - 1) + 230 * zoomScale, Math.max(height / 2 * (1 - zoomScale) - 230 * zoomScale, zoomTrans[1]));
    //  zoom.translate(zoomTrans);
    /*
    if (x.domain()[0] < xmin) {
      zoom.translate([zoom.translate()[0] - x(xmin) + x.range()[0], zoom.translate()[1]]);
    } else if (x.domain()[1] > xmax) {
      zoom.translate([zoom.translate()[0] - x(xmax) + x.range()[1], zoom.translate()[1]]);
    }*/

    var g = d3.select(this);


    //used to match future data updates zoom/translation with this one

    //update

    draw(g);

  }
  /***********************************************************


   ***********************************************************/
  chart.update = function (selection, dataSet) {

    dataSet = dataSet.map(function (d, i) {
      return [xValue.call(dataSet, d, i), yValue.call(dataSet, d, i)];
    });

    console.log(selection);

    updateXScale(dataSet);
    updateYScale(dataSet);

    zoom.x(xScale)
      .y(yScale)
      .scaleExtent([1, 10]);

    //zoom to match with what the user set
    zoom.translate(zoomTrans);
    zoom.scale(zoomScale);

    var svg = selection.select("svg");
    var g = svg.select("g");

    // Update the outer dimensions.
    svg.attr("width", width)
      .attr("height", height);

    g.data([dataSet]);

    //draw(g);

    var path = g.select(".line")
      .attr("class", "line")
      .transition()
      .ease("linear")
      .duration(1000)
      .attr("d", line);

    g.select(".area")
      .transition()
      .ease("linear")
      .duration(1000)
      .attr("d", area.y0(yScale.range()[0]));

    g.select(".x.axis")
      .transition()
      .ease("linear")
      .duration(1000)
      .attr("transform", "translate(0," + yScale.range()[0] + ")")
      .call(xAxis);

    g.select(".y.axis")
      .transition()
      .ease("linear")
      .duration(1000)
      .attr("transform", "translate(" + xScale.range()[0] + ")")
      .call(yAxis);

    return chart;
  };

  // get/set margin
  chart.margin = function (_) {
    if (!arguments.length) return margin;
    margin = _;
    return chart;
  };

  // get/set width
  chart.width = function (_) {
    if (!arguments.length) return width;
    width = _;
    return chart;
  };

  // get/set height
  chart.height = function (_) {
    if (!arguments.length) return height;
    height = _;
    return chart;
  };

  // get/set xValue function
  chart.x = function (_) {
    if (!arguments.length) return xValue;
    xValue = _;
    return chart;
  };

  // get/set yValue function
  chart.y = function (_) {
    if (!arguments.length) return yValue;
    yValue = _;
    return chart;
  };

  // get/set yValue function
  chart.drawLine = function (_) {
    if (!arguments.length) return drawLine;
    drawLine = _;
    return chart;
  };

  // get/set yValue function
  chart.drawArea = function (_) {
    if (!arguments.length) return drawArea;
    drawArea = _;
    return chart;
  };

  // get/set yValue function
  chart.drawPoints = function (_) {
    if (!arguments.length) return drawPoints;
    drawPoints = _;
    return chart;
  };

  return chart;
}
