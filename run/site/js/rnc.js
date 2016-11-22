// Rush 'n Crush example client

function RushNCrush(url, canvas_id) {
	this.canvas = document.getElementById(canvas_id);
	this.ctx = this.canvas.getContext("2d");

	// map variables
	this.map = [[]];
	this.mapw = 0;
	this.maph = 0;
	this.focux = 0;
	this.focuy = 0;
	this.playerlock = true;
	this.player_index = -1;
	this.zoom = 1;
	this.gamerunning = false;

	// users
	this.userid = undefined;
	this.user_turn = undefined;
	this.ingame = false;
	this.clickable = []; // clickable and mouseover ui elements, {x,y,w,h,click,over}
	
	// game objects
	this.players = [];
	this.team_color = ["#03c", "#c00", "#60c", "#093", "#0cc", "#fc0"]; // 6 starter color, we will randomly add more as needed (only multiples of 3)
	this.powerups = [];
	this.targets = [];

	this.setpath = [];
	this.shootmode = false;
	this.actionqueue = [];

	this.animating = false;
	this.player_ani_queue = [];

	this.ws = new WebSocket(url);
	that = this;
	this.ws.onmessage = function(evt) {
		var msg = JSON.parse(evt.data);
		console.log(msg);
		that.handle_message(msg.message_type, msg.data);
	};

	// Set up interaction
	this.canvas.addEventListener('wheel', function(evt){
		// scroll nicely by mouse location
		var rect = that.canvas.getBoundingClientRect();
		var px = event.clientX - rect.left;
		var py = event.clientY - rect.top;
		that.handle_scroll(px,py, evt.deltaY/30);
		evt.returnValue = false;
		return false; 
	}, false);
	this.canvas.addEventListener('mousedown', function(evt){
		var rect = that.canvas.getBoundingClientRect();
		var px = event.clientX - rect.left;
		var py = event.clientY - rect.top;
		that.handle_click(px, py);
	});
	this.canvas.addEventListener('mousemove', function(evt){
		var rect = that.canvas.getBoundingClientRect();
		var px = event.clientX - rect.left;
		var py = event.clientY - rect.top;
		that.handle_point(px, py);
		
	});
	document.addEventListener('keydown', function(evt) {
		var ret_code = false;
		var panspeed = 24;
		if (evt.keyCode == 65) {
			that.actionqueue = [];
			that.move_player(-1,0);
		} else if (evt.keyCode == 68) {
			that.actionqueue = [];
			that.move_player(1,0);
		} else if (evt.keyCode == 87) {
			that.actionqueue = [];
			that.move_player(0,-1);
		} else if (evt.keyCode == 83) {
			that.actionqueue = [];
			that.move_player(0,1);
		} else if (evt.keyCode == 37) {
			that.focux -= (1/that.zoom) * panspeed;
			if (that.focux < 0) {
				that.focux = 0;
			}
			that.playerlock = false;
			that.draw(true);
		} else if (evt.keyCode == 39) {
			that.focux += (1/that.zoom) * panspeed;
			if (that.focux > that.mapw) {
				that.focux = that.mapw;
			}
			that.playerlock = false;
			that.draw(true);
		} else if (evt.keyCode == 38) {
			that.focuy -= (1/that.zoom) * panspeed;
			if (that.focuy < 0) {
				that.focuy = 0;
			}
			that.playerlock = false;
			that.draw(true);
		} else if (evt.keyCode == 40) {
			that.focuy += (1/that.zoom) * panspeed;
			if (that.focuy > that.maph) {
				that.focuy = that.maph;
			}
			that.playerlock = false;
			that.draw(true);
		} else if (evt.keyCode == 32) {
			that.actionqueue = [];
			that.next_player();
		} else if (evt.keyCode == 13) {
			that.actionqueue = [];
			that.end_turn();
		} else if (evt.keyCode <= 57 && evt.keyCode >= 49) {
			that.actionqueue = [];
			that.choose_weapon(evt.keyCode - 49);
		} else {
			ret_code = true;
		}
		evt.returnValue = ret_code;
		return ret_code;
	});
}

RushNCrush.prototype.handle_scroll = function(px, py, scrolly) {
		var lockpoint = this.px2coord(px,py);
		var pfocus = this.coord2px(that.focux, that.focuy);
		// if the lock point is within 1 square of our selected guy, just lock onto the player 

		if (this.player_index >= 0 && this.players[this.player_index].pos.x == Math.floor(lockpoint[0]) && Math.floor(lockpoint[0]) == Math.floor(this.focux) && this.players[this.player_index].pos.y == Math.floor(lockpoint[1]) && Math.floor(lockpoint[1]) == Math.floor(this.focuy)) {
			this.zoom += scrolly;
			if (this.zoom <= 1) {
				this.zoom = 1
			}
			this.focux = this.players[this.player_index].pos.x + 0.5;
			this.focuy = this.players[this.player_index].pos.y + 0.5;
			this.playerlock = true;
			console.log("Locked");
		} else {
			var dx = px - pfocus[0];
			var dy = py - pfocus[1];

			this.zoom += scrolly;
			if (this.zoom <= 1) {
				this.zoom = 1
			}
			var lockpx = this.coord2px(lockpoint[0], lockpoint[1]);
			var nfocus = this.px2coord(lockpx[0] - dx, lockpx[1] - dy);
			this.focux = nfocus[0];
			this.focuy = nfocus[1];

			this.playerlock = false;
		}

		this.draw(true);
}

RushNCrush.prototype.handle_point = function(px, py) {
	if (this.animating) {
		return;
	}
	coord = that.px2coord(px, py);
	// aim, if it is your turn and you have a guy selected, and he has turns
	if (this.user_turn != undefined && this.user_turn == this.userid && this.player_index >= 0) {
		var plx = this.players[this.player_index].pos.x + 0.5;
		var ply = this.players[this.player_index].pos.y + 0.5;
		var ang = 180 * Math.atan2(coord[1] - ply, coord[0] - plx) / Math.PI;
		this.players[this.player_index].dir = Math.floor(ang);
		this.draw(false);

		var aimcursor = true;
		for (var i=0; i<this.clickable.length; i++) {
			if (px > this.clickable[i].x && py > this.clickable[i].y && px < this.clickable[i].x + this.clickable[i].w && py < this.clickable[i].y + this.clickable[i].h) {
				aimcursor = false;
				break;
			}
		}
		if (aimcursor && (coord[0] < 0 || coord[1] < 0 || coord[0] > this.mapw || coord[1] > this.maph)) {
			aimcursor = false;
		}
		
		if (aimcursor) {
			// we are pointing at the game map
			// if we are in shoot mode:
			if (this.shootmode) {
				if (document.body.style.cursor != "crosshair") {
					document.body.style.cursor = "crosshair";
				}
			} else {
				// if we are in move mode:
				// pathfinding
				this.setpath = this.pathfind(this.players[this.player_index].pos.x, this.players[this.player_index].pos.y, coord[0], coord[1], this.players[this.player_index].moves);
			}

		} else {
			if (document.body.style.cursor != "default") {
				document.body.style.cursor = "default";
			}
			this.setpath = [];
		}
	} else {
		if (document.body.style.cursor != "default") {
			document.body.style.cursor = "default";
		}
		this.setpath = [];
	}
};

RushNCrush.prototype.handle_click = function(px, py) {
	if (this.animating) {
		return;
	}
	// clear action queue
	if (this.actionqueue.length > 0) {
		this.actionqueue = [];
		console.log("Cleared action queue because of click");
	}
	// check if we are over any of our hotspots
	for (var i=0; i<this.clickable.length; i++) {
		this.draw(false);
		if (px > this.clickable[i].x && py > this.clickable[i].y && px < this.clickable[i].x + this.clickable[i].w && py < this.clickable[i].y + this.clickable[i].h) {
			this.clickable[i].click();
			return;
		}
	}
	if (this.shootmode) {
		// check bounds	
		coord = this.px2coord(px, py);
		if (coord[0] >= 0 && coord[1] >= 0 && coord[0] < this.mapw && coord[1] < this.maph) {
			this.fire();
		}
	} else if (this.setpath.length > 0) {
		// add to action queue
		for (var i=this.setpath.length-2; i>= 0; i--) {
			var that = this;
			var action = {};
			action.call = function() {
				if (this.condition()) {
					this.act();
					return true;
				}
				return false;
			};
			action.expectplayerid = this.players[this.player_index].id;
			action.expectposition = this.setpath[i+1];
			action.dx = this.setpath[i].x - this.setpath[i+1].x;
			action.dy = this.setpath[i].y - this.setpath[i+1].y;
			action.condition = function() {
				if (that.players[that.player_index].id !== this.expectplayerid) {
					return false;
				}
				if (that.players[that.player_index].pos.x !== this.expectposition.x || that.players[that.player_index].pos.y !== this.expectposition.y) {
					return false;
				}
				return true;
			};
			action.act = function() {
				that.move_player(this.dx, this.dy);
			};
			this.actionqueue.push(action);
		}
		this.continue_actionqueue();
	}
	// remove shootmode
	this.shootmode = false;
	if (document.body.style.cursor != "default") {
		document.body.style.cursor = "default";
	}
	this.setpath = [];
}

RushNCrush.prototype.continue_actionqueue = function() {
	if (this.actionqueue.length === 0) {
		return false;
	}
	var a = this.actionqueue.shift(1);
	var ret = a.call();
	if (!ret) {
		console.log("action queue condition not met");
		this.actionqueue = [];
		return false;
	}
	return true;
}

RushNCrush.prototype.fire = function() {
	if (this.animating) {
		return;
	}
	// shoot if it is your turn and you have a guy selected
	if (this.user_turn == this.userid && this.player_index >= 0) {
		var wi = this.players[this.player_index].selected_weapon
		if (wi == undefined) {
			wi = 0;
		}
		this.ws.send("fire:"+ this.players[this.player_index].id +","+ this.players[this.player_index].weapons[wi].name +","+ this.players[this.player_index].dir);
		console.log("sent fire");
	}	
};

RushNCrush.prototype.handle_message = function(message_type, data) {
	changed = false;
	if (message_type == "map") {
		changed = this.build_map(data)
	} else if (message_type == "update") {
		changed = this.update_game(data)	
	}

	if (changed == true) {
		// draw animations
		// then draw everything
		that = this;
		this.run_animations_oneatatime(function() {
			that.draw(true);
			// next action
			that.continue_actionqueue();
		});
	}
};

RushNCrush.prototype.choose_weapon = function(wi) {
	if (this.player_index >= 0 && this.players[this.player_index].weapons.length > wi) { 
		if (this.players[this.player_index].weapons[wi].ammo != 0 && this.players[this.player_index].moves >= this.players[this.player_index].weapons[wi].move_cost) {
			this.players[this.player_index].selected_weapon = wi;
			this.draw(false);
			this.shootmode = true;
		}
	}
	
};

RushNCrush.prototype.end_turn = function() {
	this.ws.send("end_turn:");
	this.setpath = [];
	console.log("sent end_turn");
};

RushNCrush.prototype.send_start = function() {
	this.ws.send("start_game:");
	console.log("sent game_start");
}

RushNCrush.prototype.move_player = function(dx, dy) {
	/*if (this.animating) {
		return;
	}*/
	if (this.player_index < 0) {
		return;
	}
	i = this.player_index;
	// check if we can move
	if (this.user_turn != this.userid) {
		return;
	}
	p = 0;
	if (this.map[this.players[i].pos.y + dy][this.players[i].pos.x + dx].tType < 8) {
		return;
	}
	// send the move
	this.ws.send("player_move:"+ this.players[i].id +","+ (this.players[i].pos.x + dx) +","+ (this.players[i].pos.y + dy) +","+ (this.players[i].dir));
	console.log("sent player_move");
};

RushNCrush.prototype.next_player = function() {
	for (var i=1; i<=this.players.length; i++) {
		if (this.players[(this.player_index + i) % this.players.length].owner == this.userid) {
			this.player_index = (this.player_index + i) % this.players.length;
			this.focux = this.players[this.player_index].pos.x + 0.5;
			this.focuy = this.players[this.player_index].pos.y + 0.5;
			this.playerlock = true;
			this.setpath = [];
			this.draw(false);
			return;
		}
	}
};

RushNCrush.prototype.choose_player = function(id) {
	for (var i=0; i<this.players.length; i++) {
		if (this.players[i].id == id) {
			this.player_index = i;
			this.focux = this.players[this.player_index].pos.x + 0.5;
			this.focuy = this.players[this.player_index].pos.y + 0.5;
			this.playerlock = true;
			this.setpath = [];
			this.draw(false);
			return;
		}
	}
};

RushNCrush.prototype.get_client_color = function(clientid) {
	var color = this.team_color[clientid];
	if (color == undefined) {
		// add a random color
		for (var i=0; i<99; i++) {
			color = "#" + (Math.floor(Math.random()*5)*3).toString(16) + (Math.floor(Math.random()*5)*3).toString(16) + (Math.floor(Math.random()*5)*3).toString(16);
			repeat = false;
			for (var j=0; j<this.team_color.length; j++) {
				if (color == this.team_color[j]) {
					repeat = true;
					break;
				}
			}
			if (!repeat) {
				break;
			}
		}
		this.team_color[clientid] = color;
	}
	return color;
};

RushNCrush.prototype.update_game = function(data) {
	this.userid = data.your_id;
	this.user_turn = data.current_turn;
	// update running
	this.gamerunning = data.game_running;
	// update page title
	if (data.your_id == data.current_turn) {
		document.title = "**YOUR TURN**";
	} else {
		document.title = "Rush n' Crush";
	}
	// update tiles
	var u_t = data.updated_tiles;
	for (var i=0; i<u_t.length; i++) {
		this.map[u_t[i].pos.y][u_t[i].pos.x] = u_t[i];
	}
	// update targets
	var u_r = data.updated_targets;
	for (var i=0; i<u_r.length; i++) {
		var found = false;
		for (var j=0; j<this.targets.length; j++) {
			if (this.targets[j].pos.x == u_r[i].pos.x && this.targets[j].pos.y == u_r[i].pos.y) {
				found = true;
				this.targets[j] = u_r[i];
			}
		}
		if (!found) {
			this.targets.push(u_r[i]);
		}
	}
	// update players
	var u_p = data.updated_players;
	if (u_p.length != 0) {
		// for every player, if updated, cool, if not, ditch 'em
		var focusid = -1;
		if (this.player_index > -1) {
			focusid = this.players[this.player_index].id;
		}
		var has_players = false;
		for (var p=0; p<this.players.length; p++) {
			p_updated = false;
			for (var i=0; i<u_p.length; i++) {
				if (u_p[i].owner == this.userid) {
					has_players = true;
				}
				if (u_p[i].id == this.players[p].id) {
					// if the player moved, animate it
					if (this.players[p].pos.x != u_p[i].pos.x || this.players[p].pos.y != u_p[i].pos.y) {
						// queue for animation
						this.player_animate(p, this.players[p].pos.x, this.players[p].pos.y, u_p[i].pos.x, u_p[i].pos.y);
					}
					// keep the correct weapon selected
					var selected_weapon = this.players[p].selected_weapon;
					// remove this from our updated array, and break
					this.players[p] = u_p.splice(i,1)[0];
					this.players[p].selected_weapon = selected_weapon;
					p_updated = true;
					break;
				}
			}
			if (!p_updated) {
				// remove the player, and adjust
				this.players.splice(p,1);
				// reset focus
				this.player_index = -1;
				if (focusid >= 0) {
					for (var np=0; np < this.players.length; np++) {
						if (this.players[np].id == focusid) {
							if (this.playerlock) {
								this.focux = this.players[np].pos.x + 0.5;
								this.focuy = this.players[np].pos.y + 0.5;
							}
							this.player_index = np;
							break;
						}
					}
				}
				p--;
			}
		}
		this.ingame = has_players;
		// if there are extras left over, add them
		for (var i=0; i<u_p.length; i++) {
			// saw a new player, clear the queue
			if (this.actionqueue.length > 0) {
				this.actionqueue = [];
				console.log("Cleared action queue, because we saw an enemy");
			}
			this.players.push(u_p[i]);
			if (u_p[i].owner == this.userid && this.player_index < 0) {
				// focus on the new player
				this.focux = u_p[i].pos.x + 0.5;
				this.focuy = u_p[i].pos.y + 0.5;
				this.playerlock = true;
				this.player_index = this.players.length - 1;
			}
		}
	}
	// handle powerups
	this.powerups = data.powerups;

	// handle hit tiles
	var h_t = data.hit_tiles;
	for (var i=0; i<h_t.length; i++) {
		this.hit_animate(h_t[i].pos.x, h_t[i].pos.y, h_t[i].from_pos.x, h_t[i].from_pos.y, h_t[i].damage_type);
	}

	return true;
};

RushNCrush.prototype.build_map = function(maparr) {
	this.map = maparr;
	this.maph = maparr.length;
	this.mapw = maparr[0].length;	

	// set default focus
	this.focux = this.mapw / 2.0;
	this.focuy = this.maph / 2.0;
	this.playerlock = false;

	// set default zoom
	this.zoom = this.canvas.width / this.mapw;
	if (this.canvas.height / this.maph < this.zoom) {
		this.zoom = this.canvas.height / this.maph;
	}

	return true;
};

RushNCrush.prototype.run_animations = function(callback) {
	that = this;
	this.animating = true;
	var animate = function(dt) {
		that.draw(false);
		if (that.player_ani_queue.length == 0) {
			that.animating = false;
			callback();
			return;
		}else {
			for (var i=0; i<that.player_ani_queue.length; i++) {
				if (that.player_ani_queue[i]() == false) {
					that.player_ani_queue.splice(i,1);
					i--;
				}
			}
		}
		window.requestAnimationFrame(animate);
	}
	animate();
}

RushNCrush.prototype.run_animations_oneatatime = function(callback) {
	that = this;
	this.animating = true;
	var animate = function(dt) {
		that.draw(false);
		if (that.player_ani_queue.length == 0) {
			that.animating = false;
			callback();
			return;
		}
		var ani_group = that.player_ani_queue[0];
		if (ani_group.length === 0) {
			that.player_ani_queue.shift(1);
		} else {
			for (var i=0; i<ani_group.length; i++) {
				if (ani_group[i]() == false) {
					ani_group.splice(i,1);
					i--;
				}
			}
		}
		window.requestAnimationFrame(animate);
	}
	animate();
}

RushNCrush.prototype.hit_animate = function(hitx,hity, fromx,fromy, type) {
	var steps = 9;
	var anistep = 1;
	var that = this;
	var topl = this.coord2px(hitx,hity);
	var tx1 = topl[0] + (this.zoom/2);
	var ty1 = topl[1] + (this.zoom/2);
	var from_topl = this.coord2px(fromx,fromy);
	var tx2 = from_topl[0] + (this.zoom/2);
	var ty2 = from_topl[1] + (this.zoom/2);
	var w = this.zoom;
	var draw_hit = function() {
		if (anistep >= steps) {
			return false;
		}
		// block hit
		var fade = 1.0 / anistep + 0.2;
		var pad = -0.5;
		that.ctx.fillStyle = "rgba(200,0,0,"+ fade +")";
		that.ctx.fillRect(topl[0] + pad, topl[1] + pad, w - (pad * 2), w - (pad * 2));
		// shot trace
		that.ctx.strokeStyle = "rgba(255,263,33,"+ fade +")";
		that.ctx.linewidth = 9;
		that.ctx.beginPath();
		that.ctx.moveTo(tx1, ty1);
		that.ctx.lineTo(tx2, ty2);
		that.ctx.stroke();
		anistep++;
		return true;
	}
	if (this.player_ani_queue.length === 0) {
		this.player_ani_queue[0] = [];
	}
	this.player_ani_queue[this.player_ani_queue.length - 1].push(draw_hit);
}

RushNCrush.prototype.player_animate = function(p_index, sx,sy, x,y) {
	var steps = 6;
	var anistep = 1;
	var that = this;
	var update_player = function() {
		if (that.players[p_index] === undefined) {
			console.log("Lost one during animation")
			return false
		}
		// change player location and dir
		dx = (x - sx) * (anistep / steps);
		dy = (y - sy) * (anistep / steps);
		that.players[p_index].pos.x = sx + dx;
		that.players[p_index].pos.y = sy + dy;

		// change focus
		if (p_index == that.player_index && that.playerlock) {
			that.focux = sx + dx + 0.5;
			that.focuy = sy + dy + 0.5;
		}
		anistep = anistep + 1;

		if (anistep > steps) {
			anistep = 0;
			return false
		} else {
			return true
		}
	}
	// put this function into the queue
	this.player_ani_queue.push([update_player]);
}

// Turns an x,y coordinate to the pixel coordinate at the top left of the box
// returns a array with [px, py]
RushNCrush.prototype.coord2px = function(x, y) {
	var px;
	var py;
	
	px = ((x * this.zoom) - ((this.focux) * this.zoom)) + (this.canvas.width / 2);
	py = ((y * this.zoom) - ((this.focuy) * this.zoom)) + (this.canvas.height / 2);

	return [px, py];
};

RushNCrush.prototype.px2coord = function(px, py) {
	var x;
	var y;

	x = ((px - (this.canvas.width / 2)) / this.zoom) + (this.focux);
	y = ((py - (this.canvas.height / 2)) / this.zoom) + (this.focuy);

	return [x,y];
};

RushNCrush.prototype.draw = function(cast) {
	// clear clickable area
	this.clickable = [];
	// if the canvas size is wrong, set it
	
	if (cast && (this.ctx.canvas.width != window.innerWidth || this.ctx.canvas.height != window.innerHeight)) {
		this.ctx.canvas.width  = window.innerWidth - 12;
		this.ctx.canvas.height = window.innerHeight - 12;
	}
	// Clear
	this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
	
	if (cast) {
		this.ray_cast_clear();
		// Mark things for Shadow
		for (var i=0; i<this.players.length; i++) {
			if (this.players[i].owner == this.userid) {
				this.ray_cast_start(this.players[i].pos.x, this.players[i].pos.y);
			}
		}
	}

	// Draw the tiles
	for (var x=0; x<this.mapw; x++) {
		for (var y=0; y<this.maph; y++) {
			this.draw_tile(this.map[y][x], x, y);
		}
	}

	// Draw powerups
	for (var i=0; i<this.powerups.length; i++) {
		this.draw_powerup(this.powerups[i]);
	}

	// Draw the path
	if (!this.shootmode) {
		this.draw_path(this.setpath);
	}

	// Draw the players
	for (var i=0; i<this.players.length; i++) {
		this.draw_player(this.players[i]);
	}

	// Draw the targets
	for (var i=0; i<this.targets.length; i++) {
		this.draw_target(this.targets[i]);
	}

	// Draw the UI
	this.draw_ui();
};

RushNCrush.prototype.draw_player = function(player) {
	if (player.pos.x < 0 || player.pos.y < 0) {
		return;
	}
	var center = this.coord2px(player.pos.x + 0.5, player.pos.y + 0.5);
	var teamcolor = this.get_client_color(player.owner);
	this.ctx.fillStyle = teamcolor;
	this.ctx.strokeStyle = "#000000";
	this.ctx.beginPath();
	this.ctx.arc(center[0], center[1], this.zoom/2 - 0.5, 0, 2*Math.PI);
	this.ctx.fill();
	if (this.player_index >= 0 && this.players[this.player_index].id == player.id) {
		this.ctx.lineWidth = this.zoom/18;
		this.ctx.stroke();
	}
	// draw direction piece
	this.ctx.fillStyle = "#FFFFFF";
	this.ctx.lineWidth = this.zoom / 90;
	this.ctx.beginPath();
	this.ctx.moveTo(center[0], center[1]);
	this.ctx.arc(center[0], center[1], this.zoom/2 - 0.5, (Math.PI * player.dir/180) - Math.PI/18, (Math.PI * player.dir/180) + Math.PI/18);
	this.ctx.lineTo(center[0], center[1]);
	this.ctx.fill();
	this.ctx.stroke();
	if (player.owner == this.userid) {
		// draw health
		this.ctx.fillStyle = teamcolor;
		var bottoml = this.coord2px(player.pos.x, player.pos.y + 1);
		var height = this.zoom / 15;
		var width = this.zoom * (player.health / player.max_health);
		this.ctx.fillRect(bottoml[0], bottoml[1] + height, width, height);

		// add clickable area
		var that = this;
		var clickarea = {x:center[0] - (this.zoom/2), y:center[1] - (this.zoom/2), w:this.zoom, h:this.zoom, id:player.id, click: function() {
			that.choose_player(this.id);
		}, over: function() {}};
		this.clickable.push(clickarea);
	}
};

RushNCrush.prototype.draw_tile = function(tile_obj, x, y) {
	var no_draw = false;
	var circle = false;
	switch (tile_obj.tType) {
	case 1:
		this.ctx.fillStyle = "#000000";
		// Invincible wall
		break;
	case 2:
		this.ctx.fillStyle = "#5c5c5c";
		// Strong wall
		break;
	case 3:
		this.ctx.fillStyle = "#4a301b";
		// Weak wall
		break;
	case 4:
		this.ctx.fillStyle = "#b0b0b0";
		// Strong vertical cover
		break;
	case 5:
		this.ctx.fillStyle = "#b0b0b0";
		// Strong horizontal cover
		break;
	case 6:
		this.ctx.fillStyle = "#b39b82";
		// Weak vertical cover
		break;
	case 7:
		this.ctx.fillStyle = "#b39b82";
		// Weak horizontal cover
		break;
	case 8:
		no_draw = true;
		// Walkable tile
		break;
	case 9:
		no_draw = true;
		// Spawn
		break;
	case 10:
		no_draw = true;
		// Flag
		break;
	case 11:
	case 12:
	case 13:
		no_draw = true;
		circle = true;
		this.ctx.strokeStyle = "#daa520";
		// pup
		break;
	case 14:
		no_draw = true;
		// target spawn
		break;
	}
	
	var topl = this.coord2px(x,y);
	var w = this.zoom;
	var pad = -0.06;
	// Draw tile
	if (no_draw == false) {
		this.ctx.fillRect(topl[0] + pad, topl[1] + pad, w - (2*pad), w - (2*pad));
	}
	if (circle == true) {
		this.ctx.beginPath();
		this.ctx.arc(topl[0]+(w/2), topl[1]+(w/2), w * 0.4, 0, 2*Math.PI);
		this.ctx.stroke();
	}
	// Draw shadow
	if (!tile_obj.lit && this.ingame) {
		this.ctx.fillStyle = "rgba(0,0,0,0.45)";
		this.ctx.fillRect(topl[0] + pad, topl[1] + pad, w - (2*pad), w - (2*pad));
	}

	// draw debug index
	//this.ctx.fillStyle = "#000000";
	//this.ctx.font="6px";
	//this.ctx.fillText(""+x+","+y, topl[0] + 3, topl[1] + (w/2));
};

RushNCrush.prototype.draw_powerup = function(powerup) {
	var x = powerup.pos.x;
	var y = powerup.pos.y;
	this.ctx.fillStyle = "#daa520";
	var rad = this.zoom * 0.4;
	var center = this.coord2px(x + 0.5, y + 0.5);
	this.ctx.beginPath();
	this.ctx.moveTo(center[0], center[1] - rad);
	var spikes = 5;
	var angstep = 4 * Math.PI / spikes;
	for (var i=0; i < spikes; i++) {
		var dx = Math.sin(angstep * i) * rad;
		var dy = Math.cos(angstep * i) * rad;
		this.ctx.lineTo(center[0] - dx, center[1] - dy);
	}
	this.ctx.fill();
};

RushNCrush.prototype.draw_target = function(target) {
	if (target.health <= 0) {
		return;
	}
	var center = this.coord2px(target.pos.x + 0.5, target.pos.y + 0.5);
	var teamcolor = this.get_client_color(target.owner);
	this.ctx.strokeStyle = teamcolor;
	this.ctx.fillStyle = teamcolor;

	this.ctx.beginPath();
	this.ctx.lineWidth = this.zoom/6;
	this.ctx.arc(center[0], center[1], (this.zoom/2) - (this.ctx.lineWidth / 2), 0, 2*Math.PI);
	this.ctx.stroke();

	this.ctx.beginPath();
	this.ctx.arc(center[0], center[1], this.zoom/6, 0, 2*Math.PI);
	this.ctx.fill();
};

RushNCrush.prototype.draw_path = function(path) {
	if (path == undefined || path.length === 0) {
		return;
	}
	var teamcolor = this.get_client_color(this.userid);
	this.ctx.strokeStyle = teamcolor;
	this.ctx.lineWidth = this.zoom / 9;
	this.ctx.beginPath();
	for (var i=0; i<path.length; i++) {
		var center = this.coord2px(path[i].x + 0.5, path[i].y + 0.5);
		if (i===0) {
			this.ctx.moveTo(center[0], center[1]);
		} else {
			this.ctx.lineTo(center[0], center[1]);
		}
	}
	this.ctx.stroke();
	
}

RushNCrush.prototype.draw_ui = function() {
	var box_size = 12;
	var pad = 4;

	// Moves and Current Turn
	this.ctx.fillStyle = this.team_color[this.user_turn];
	this.ctx.strokeStyle = "#000";
	this.ctx.lineWidth = 1;

	var player_moves = 1;
	if (this.userid == this.user_turn && this.player_index >= 0) {
		player_moves = this.players[this.player_index].moves;
	}
	for (var i=0; i<player_moves; i++) {
		var x = pad;
		var y = (i * (pad + box_size)) + pad;
		this.ctx.fillRect(x,y, box_size,box_size);
		if (i == 0) {
			this.ctx.strokeRect(x - pad/2,y - pad/2, box_size + pad, box_size + pad);
		}
	}

	// Weapons
	if (this.player_index < 0) {
		return;
	}
	this.ctx.fillStyle = "#FFF";
	this.ctx.font = "12px Verdana";
	var text_height = this.ctx.measureText("M").width;
	var box_size = 2 * text_height;
	for (var i=0; i<this.players[this.player_index].weapons.length; i++) {
		var ammo = this.players[this.player_index].weapons[i].ammo;
		var move_cost = this.players[this.player_index].weapons[i].move_cost;
		var text = this.players[this.player_index].weapons[i].name +" : "+ move_cost +" : "+ ((ammo >= 0)?ammo:String.fromCharCode(8734));
		var w = this.ctx.measureText(text).width + (2 * pad);
		var x = this.canvas.width - (pad + w);
		var y = (i * (pad + box_size)) + pad;
		this.ctx.fillStyle = "#FFF";
		if (ammo == 0 || player_moves < move_cost) {
			this.ctx.fillStyle = "#999"
		}
		this.ctx.fillRect(x, y, w, box_size);

		var wi = this.players[this.player_index].selected_weapon
		if (!this.shootmode || wi == undefined) {
			wi = 0;
		} else if (i == wi) {
			this.ctx.strokeRect(x - pad/2,y - pad/2, w + pad, box_size + pad);
		}

		this.ctx.fillStyle = "#000";
		if (ammo == 0 || player_moves < move_cost) {
			this.ctx.fillStyle = "#555"
		}
		this.ctx.fillText(text, x + pad, y + (box_size * 0.75));
		// add clickable area
		var that = this;
		var clickarea = {x:x, y:y, w:w, h:box_size, index:i, click: function() {
			that.choose_weapon(this.index);
		}, over: function() {}};
		this.clickable.push(clickarea);
	}

	// start game
	if (!this.gamerunning) {
		var w = this.canvas.width / 4;
		var h = this.canvas.height / 6;
		var text = "START GAME";
		this.ctx.font = "24px Verdana";
		var tw = this.ctx.measureText(text).width;
		var th = this.ctx.measureText("M").width;
		this.ctx.fillStyle = "rgba(255,255,255,0.96)";
		this.ctx.strokeStyle = "#000";
		this.ctx.fillRect((this.canvas.width / 2) - (w/2), (this.canvas.height * 0.6) - (h/2), w, h);
		this.ctx.strokeRect((this.canvas.width / 2) - (w/2), (this.canvas.height * 0.6) - (h/2), w, h);
		this.ctx.fillStyle = "#000";
		this.ctx.fillText(text, (this.canvas.width / 2) - (tw/2), (this.canvas.height * 0.6) + (th/2));
		var that = this;
		var clickarea = {x:(this.canvas.width / 2) - (w/2), y:(this.canvas.height * 0.6) - (h/2), w:w, h:h, click: function() {
			that.send_start();
		}, over: function() {}};
		this.clickable.push(clickarea);
	}
};

RushNCrush.prototype.tile_moveable = function(mx,my) {
	if (mx < 0 || mx >= this.mapw || my < 0 || my >= this.maph) {
		return false;
	}
	tt = this.map[my][mx].tType;
	if (tt <= 7 || tt == 14) { // a bit of a cheat, because 14 can we walked through if the target is shot
		return false;
	}
	// check if anyone is there
	for (var i=0; i<this.players.length; i++) {
		if (this.players[i].pos.x == mx && this.players[i].pos.y == my) {
			return false;
		}
	}
	return true;
}

RushNCrush.prototype.pathfind = function(sx,sy, ex,ey, maxdepth) {
	ex = Math.floor(ex);
	ey = Math.floor(ey);
	// check if we are trying to move to a taken spot
	if (!this.tile_moveable(ex,ey)) {
		return [];
	}
	var fmap = [[]];
	var q = [];
	fmap[sy] = [];
	fmap[sy][sx] = {px:sx, py:sy}; // add to seen map
	q.push({x:sx, y:sy, depth:0}); // add to queue
	var reached = false;
	while (q.length > 0) {
		// check if we made it
		if (q[0].x == ex && q[0].y == ey) {
			reached = true;
			break;
		} else if (q[0].depth < maxdepth) {
			// add neighbors
			for (var dy=-1; dy<=1; dy++) {
				for (var dx=-1; dx<=1; dx++) {
					// no diagonal
					if (dy !== 0 && dx !== 0) {
						continue;
					}
					if (dy === 0 && dx === 0) {
						continue;
					}
					var mx = q[0].x + dx;
					var my = q[0].y + dy;
					// make sure we haven't checked it already
					if (fmap[my] !== undefined && fmap[my][mx] !== undefined) {
						continue;
					}
					// check walkable
					if (!this.tile_moveable(mx,my)) {
						continue;
					}
					// possible loop through targets here
					// Passed walkable
					// add its previous
					if (fmap[my] === undefined) {
						fmap[my] = [];
					}
					fmap[my][mx] = {px: q[0].x, py: q[0].y};
					// add to q
					q.push({x:mx, y:my, depth:q[0].depth+1});
				}
			}
		}
		q.shift(1);
	}
	if (reached) {
		//make the path
		var path = [];
		var px = ex;
		var py = ey;
		while (true) {
			if (px == sx && py == sy) {
				path.push({x:px, y:py});
				break;
			}
			path.push({x:px, y:py});
			var next = fmap[py][px];
			px = next.px;
			py = next.py;
		}
		return path;
	}
	return [];
}

RushNCrush.prototype.ray_cast_clear = function() {
	// clear all the tiles
	for (var x=0; x<this.mapw; x++) {
		for (var y=0; y<this.maph; y++) {
			this.map[y][x].lit = false;
		}
	}
};

RushNCrush.prototype.ray_cast_start = function(origin_x, origin_y) {
	num_cast = 256;
	for (var i=0; i<num_cast; i++) {
		var sin = Math.sin(Math.PI * 2 * (i / num_cast));
		var cos = Math.cos(Math.PI * 2 * (i / num_cast));
		var ex = cos * 60;
		var ey = sin * 60;
		this.ray_cast(origin_x, origin_y, ex + origin_x, ey + origin_y);
	}	
}

RushNCrush.prototype.ray_cast = function(px, py, ex, ey) {
	var dirx, diry;
	dirx = (ex - px > 0) ? -1 : 1;
	diry = (ey - py > 0) ? -1 : 1;
	var dx = Math.abs(ex - px);
	var dy = Math.abs(ey - py);

	var sx = px;
	var sy = py;
	var n = dx + dy;
	var err = dx - dy;
	dx *= 2;
	dy *= 2;

	for (; n >= 0; n--) {
		if (sx < 0 || sx >= this.mapw || sy < 0 || sy >= this.maph) {
			return;
		}
		this.map[sy][sx].lit = true;
		tt = this.map[sy][sx].tType;
		if (tt == 1 || tt == 2 || tt == 3) {
			return; // no walk or see
		}

		if (err > 0) {
			sx = sx + dirx;
			err = err - dy;
		} else {
			sy = sy + diry;
			err = err + dx;
		}
	}

};

var game = new RushNCrush("ws://"+ document.domain +":12345/", "gamecanvas");
