function lineChart() {

  var margin = {
      top: 20,
      right: 20,
      bottom: 20,
      left: 40
    },
    width = 1000,
    height = 500,
    xValue = function (d) {
      return d[0];
    },
    yValue = function (d) {
      return d[1];
    },
    xScale = d3.time.scale(),
    yScale = d3.scale.linear(),
    xAxis = d3.svg.axis().scale(xScale).orient("bottom").tickSize(6, 0),
    yAxis = d3.svg.axis().scale(yScale).orient("left").ticks(20),
    area = d3.svg.area().x(X).y1(Y).defined(function(d) { return !isNaN(d[1]); }),
    line = d3.svg.line().interpolate("linear").x(X).y(Y).defined(function(d) { return !isNaN(d[1]); }),
    zoom = d3.behavior.zoom().x(xScale).y(yScale).on("zoom", zoomed),
    drawLine = true, //line or no line?
    drawArea = true, //area graph or no?
    drawPoints = true, //points or no?
    drawXAxis = true,
    drawYAxis = true;

  var zoomTrans = [0, 0];
  var zoomScale = 1;

  /***********************************************************


   ***********************************************************/
  function chart(selection) {


    selection.each(function (data) {

      //console.log(this);

      //var bbox = this.getBBox();
      //width = this.offsetWidth; //bbox.width;
      //height = bbox.height;

      // Convert data to standard representation greedily;
      // this is needed for nondeterministic accessors.
      data = data.map(function (d, i) {
        return [xValue.call(data, d, i), yValue.call(data, d, i)];
      });

      updateXScale(data);
      updateYScale(data);



      // Select the svg element, if it exists.
      var svg = d3.select(this)
        .selectAll("svg")
        .data([data]);

      //update zoom scale values
      zoom.x(xScale)
        .y(yScale)
        .scaleExtent([1, 10]);


      // Otherwise, create the skeletal chart.
      var gEnter = svg.enter()
        .append("svg")
        .attr("width", '100%')
        .attr("height", '100%')
        .attr('viewBox', '0 0 ' + Math.min(width, height) + ' ' + Math.min(width, height))
        .attr('preserveAspectRatio', 'xMinYMin')
        .append("g")
        .attr("transform", "translate(" + Math.min(width, height) / 2 + "," + Math.min(width, height) / 2 + ")");;

      gEnter.append("clipPath")
        .attr("id", "clip")
        .append("rect")
        .attr("x", 0)
        .attr("y", 0)
        .attr("width", width - margin.left - margin.right)
        .attr("height", height - margin.top - margin.bottom);

      gEnter.append("rect")
        .attr("width", width - margin.left - margin.right)
        .attr("height", height - margin.top - margin.bottom)
        .attr("fill", "white");

      if (drawArea) gEnter.append("path").attr("class", "area").attr("clip-path", "url(#clip)");
      if (drawLine) gEnter.append("path").attr("class", "line").attr("clip-path", "url(#clip)");
      if (drawXAxis) gEnter.append("g").attr("class", "x axis");
      if (drawYAxis) gEnter.append("g").attr("class", "y axis");


      // Update the outer dimensions.
      svg.attr("width", width)
        .attr("height", height);

      // Update the inner dimensions.
      var g = svg.select("g")
        .attr("transform", "translate(" + margin.left + "," + margin.top + ")")
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
    });
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


  /***********************************************************


   ***********************************************************/
  function zoomed() {
    //console.log("translate: " + d3.event.translate);
    //console.log("scale: " + d3.event.scale);

    var g = d3.select(this);

    zoomScale = d3.event.scale;
    zoomTrans = d3.event.translate;
    //update

    // Update the x-axis.
    g.select(".x.axis")
      .attr("transform", "translate(0," + yScale.range()[0] + ")")
      .call(xAxis);

    g.select(".y.axis")
      .call(yAxis);

    g.select(".line")
      .attr("class", "line")
      .attr("d", line);

    g.select(".area")
      .attr("d", area.y0(yScale.range()[0]));


  }
  /***********************************************************


   ***********************************************************/
  chart.update = function (selection, dataSet) {

    dataSet = dataSet.map(function (d, i) {
      return [xValue.call(dataSet, d, i), yValue.call(dataSet, d, i)];
    });

    //console.log(dataSet);

    updateXScale(dataSet);
    updateYScale(dataSet);

    zoom.x(xScale)
      .y(yScale)
      .scaleExtent([1, 10]);

    zoom.translate(zoomTrans);
    zoom.scale(zoomScale);

    var g = selection.select("svg").select("g");

    g.data([dataSet]);

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
