(function (window, document) {

  var streamID = "";
  var currentNetwork = window.location.pathname;

  d3.select('#create-stream').on('click', function() {
    d3.json('/api/v1' + currentNetwork + '/streams')
    .header("Content-Type", "application/json")
    .post(JSON.stringify({}), function(error, data) {
      //request server error
      if (error) {
        console.log(error);
        return;
      }
      loadStream();
    });
  });

  var socketAttempts = 1;
  var JSONData;
  var margin = {
    top: 20,
    right: 20,
    bottom: 20,
    left: 40
  }

  var chart = lineChart()
    .x(function (d) {
      return new Date(d.timeStamp);
    })
    .y(function (d) {
      return +d.value;
    })
    .margin(margin);

    loadStream();

/***********************************************************


 ***********************************************************/
  function loadStream(){

      d3.json('/api/v1' + currentNetwork + '/streams', function (error, data) {

        //request server error
        if (error) {
          console.log(error);
          return;
        }

        console.log(JSON.stringify(data));

        //no streams data. Do nothing.
        if (!data.length) {
          return;
        }

        streamID = data[0].streamId;
        console.log(streamID);

        d3.json('/api/v1' + currentNetwork + '/streams/' + streamID + '/data', function (error, data) {

          if (error) {
            console.log(error);
          } else {

            console.log(JSON.stringify(data));

            JSONData = data;
            JSONData.sort(function (a, b) {
              return d3.ascending(a.timeStamp, b.timeStamp);
            }); //sort data based on time



            d3.select("#chart")
              .datum(JSONData)
              .call(chart);
          }
        });


        console.log(currentNetwork);
        url = 'ws://localhost:8000/api/v1' + currentNetwork + '/streams/' + streamID + '/socket';
        createWebSocket(url);


      });

  }



  /***********************************************************


   ***********************************************************/
  function createWebSocket(url) {
    var connection = new WebSocket(url);

    connection.onopen = function () {
      // reset the tries back to 1 since we have a new connection opened.
      socketAttempts = 1;
      socketStatus(true);

    }

    connection.onclose = function () {

      socketStatus(false);
      var time = generateInterval(socketAttempts);

      setTimeout(function () {
        // We've tried to reconnect so increment the attempts by 1
        socketAttempts++;

        // Connection has closed so try to reconnect every 10 seconds.
        createWebSocket(url);
      }, time);
    }

    connection.onmessage = function (msg) {
      var data = JSON.parse(msg.data);
      JSONData.push(data);

      chart.update(JSONData);

      var log = data.timeStamp + " value:" + data.value + '\r\n';
      d3.select("#output").node().value += log;

      console.log(msg);
    }

  }

  /***********************************************************


   ***********************************************************/
  function socketStatus(isOpen) {

    if (isOpen) {
      d3.select("#status")
        .style('color', 'green')
        .text('Server Connected');
    } else {
      d3.select("#status")
        .style('color', 'gray')
        .text('Server Disconnected');
    }

  }

  /***********************************************************


   ***********************************************************/
  function generateInterval(k) {
    var maxInterval = (Math.pow(2, k) - 1) * 1000;

    if (maxInterval > 30 * 1000) {
      maxInterval = 30 * 1000; // If the generated interval is more than 30 seconds, truncate it down to 30 seconds.
    }

    // generate the interval to a random number between 0 and the maxInterval determined from above
    return Math.random() * maxInterval;
  }


})(this, this.document);
